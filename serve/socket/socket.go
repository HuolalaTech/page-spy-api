package socket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	eventApi "github.com/HuolalaTech/page-spy-api/api/event"
	roomApi "github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/logger"
	"github.com/HuolalaTech/page-spy-api/metric"
	"github.com/HuolalaTech/page-spy-api/room"
	"github.com/HuolalaTech/page-spy-api/serve/common"
	"github.com/HuolalaTech/page-spy-api/util"
	"github.com/gorilla/websocket"
)

var joinLog = logger.Log().WithField("module", "socket")

func writeResponse(w http.ResponseWriter, res *common.Response) {
	if res.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	bs, err := json.Marshal(res)
	if err != nil {
		joinLog.WithError(err).Error("write message error")
	}

	_, err = w.Write(bs)
	if err != nil {
		joinLog.WithError(err).Error("write message error")
	}
}

type socket struct {
	rwLock sync.RWMutex
	conn   *websocket.Conn
}

func (s *socket) WriteMessage(messageType int, data []byte) error {
	return s.conn.WriteMessage(messageType, data)
}

func (s *socket) WriteDataIgnoreError(data interface{}) {
	err := s.WriteData(data)
	if err != nil {
		joinLog.WithError(err).Error("send message write message")
		s.writeWebsocketError(roomApi.NewNetWorkTimeoutError(err.Error()))
	}
}

func (s *socket) WriteData(data interface{}) error {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()
	bs, err := json.Marshal(data)
	if err != nil {
		return roomApi.NewMessageContentError("send message marshal error %s", err.Error())
	}

	return s.conn.WriteMessage(websocket.TextMessage, bs)
}

func (s *socket) WriteJSON(v interface{}) error {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()
	return s.conn.WriteJSON(v)
}

func (s *socket) ReadJSON(v interface{}) error {
	return s.conn.ReadJSON(v)
}

func (s *socket) writeWebsocketError(errRes error) {
	message := NewErrorMessage(errRes)
	if message == nil {
		return
	}

	err := s.WriteJSON(message)
	if err != nil {
		joinLog.WithError(err).Error("write websocket  message error")
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *WebSocket) readClientMessage(ctx context.Context, socket *socket, room roomApi.RemoteRoom) error {
	if room.IsClose() {
		return roomApi.NewRoomCloseError("room %s is already close", room.GetRoomAddress().ID)
	}

	rawMsg := &roomApi.RawMessage{}
	err := socket.ReadJSON(rawMsg)
	if err != nil {
		return roomApi.NewRoomCloseError("read message websocket error %s", err.Error())
	}

	msg, err := rawMsg.ToMessage()

	if err != nil {
		socket.writeWebsocketError(roomApi.NewClientError("message transform failed, %s", err))
		return nil
	}

	if !roomApi.IsPublicMessageType(msg.Type) {
		socket.writeWebsocketError(roomApi.NewClientError("message type %s is not supported to be sent by frontend", msg.Type))
		return nil
	}

	log.Debugf("socket received %s", msg.Type)
	metric.Count("server_read_message", map[string]string{
		"type": msg.Type,
	}, 1)
	switch msg.Type {
	case roomApi.UpdateRoomInfoType:
		updateRoomInfoContent := msg.Content.(*roomApi.UpdateRoomInfoContent)
		if updateRoomInfoContent.Info == nil {
			socket.writeWebsocketError(fmt.Errorf("update room info content info is nil"))
			return nil
		}

		updateRoomInfoContent.Info.Address = room.GetRoomAddress()
		info, err := s.roomManager.UpdateRoomOption(ctx, updateRoomInfoContent.Info)
		updateRoomInfoContent.Info = info
		if err != nil {
			socket.writeWebsocketError(err)
			return nil
		}

		msg.Content = updateRoomInfoContent
		socket.WriteDataIgnoreError(msg)
		return nil
	case roomApi.PingType:
		socket.WriteDataIgnoreError(msg.GetPong())
		return nil
	default:
		err = room.SendMessage(ctx, msg)
		if err != nil {
			socket.writeWebsocketError(err)
			return nil
		}

	}

	return nil
}

func onRoomMessage(ctx context.Context, socket *socket, room roomApi.RemoteRoom) error {
	select {
	case msg := <-room.OnMessage():
		now := util.TimeToNumber(time.Now())
		metric.Time("server_send_message", map[string]string{
			"type": msg.Type,
		}, float64(now-msg.CreatedAt))
		socket.WriteDataIgnoreError(msg)
	case <-room.Done():
		return roomApi.NewRoomCloseError("room %s left", room.GetRoomAddress().ID)
	case <-ctx.Done():
		socket.writeWebsocketError(roomApi.NewNetWorkTimeoutError("room %s context cancel", room.GetRoomAddress().ID))
		return nil
	}

	return nil
}

func (s *WebSocket) serveRoom(opt *roomApi.Info, connection *roomApi.Connection, socket *socket, room roomApi.RemoteRoom) {
	retCode := "success"
	close := func() {
		err := s.roomManager.LeaveRoom(context.Background(), opt, connection)
		if err != nil {
			joinLog.Errorf("serveRoom %s close %v code %s", opt.Address.ID, err, retCode)
		}
		room.Close(context.Background(), retCode)
	}

	cancelCtx, cancel := context.WithCancel(context.Background())

	metric.Count("tunnel_room", map[string]string{
		"action": "join",
		"code":   retCode,
	}, 1)

	defer func() {
		cancel()
		metric.Count("tunnel_room", map[string]string{"action": "close", "code": retCode}, 1)
		close()
	}()

	socket.conn.SetCloseHandler(func(code int, text string) error {
		cancel()
		retCode = "remote_close"
		return nil
	})

	go func() {
		writeCode := "success"
		defer func() {
			cancel()
			metric.Count("tunnel_room", map[string]string{
				"action": "close",
				"code":   writeCode,
			}, 1)

			if err := recover(); err != nil {
				retCode = "panic_close"
				joinLog.Error("serve connection panic", connection.Address.ID, err, string(debug.Stack()))
			}
		}()

		for {
			select {
			case <-cancelCtx.Done():
				return
			case <-room.Done():
				writeCode = "room_close"
				return
			default:
				err := onRoomMessage(cancelCtx, socket, room)
				if err != nil {
					writeCode = "write_message_close"
					socket.writeWebsocketError(err)
					joinLog.WithField("connection", connection.Address.ID).Info(err)
					return
				}
			}
		}
	}()

	for {
		select {
		case <-cancelCtx.Done():
			return
		case <-room.Done():
			retCode = "room_close"
			return
		default:
			err := s.readClientMessage(cancelCtx, socket, room)
			if err != nil {
				retCode = "read_message_close"
				socket.writeWebsocketError(err)
				joinLog.WithField("readClientMessage", connection.Address.ID).Info(err)
				return
			}
		}
	}

}

func NewWebSocket(rooManager *room.RemoteRpcRoomManager) *WebSocket {
	return &WebSocket{
		roomManager: rooManager,
	}
}

type WebSocket struct {
	roomManager *room.RemoteRpcRoomManager
}

type ListRoomParams struct {
	Group *string           `json:"group"`
	Name  *string           `json:"name"`
	Tags  map[string]string `json:"tags"`
}

func (s *WebSocket) ListRooms(rw http.ResponseWriter, r *http.Request) {
	tags := getTags(r.URL.Query(), "")
	rooms, err := s.roomManager.ListRooms(r.Context(), tags)
	if err != nil {
		writeResponse(rw, common.NewErrorResponse(err))
		return
	}

	writeResponse(rw, common.NewSuccessResponse(rooms))
}

func getTags(query url.Values, prefix string) map[string]string {
	tags := make(map[string]string, len(query))
	for k, v := range query {
		if len(v) > 0 {
			value := v[0]
			if prefix != "" {
				if strings.HasPrefix(k, prefix) {
					tags[k[len(prefix):]] = value
				}

			} else {
				tags[k] = value
			}
		}
	}

	return tags
}

type RoomOptions struct {
	Secret    string `json:"secret"`
	UseSecret bool   `json:"useSecret"`
}

func (s *WebSocket) CreateRoom(rw http.ResponseWriter, r *http.Request) {
	address := s.roomManager.AddressManager.GeneratorRoomAddress()
	name := r.URL.Query().Get("name")
	group := r.URL.Query().Get("group")
	tags := getTags(r.URL.Query(), "")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeResponse(rw, common.NewErrorResponse(errors.New("name or group missing")))
		return
	}
	secretOpt := &RoomOptions{}
	if len(body) > 0 {
		err = json.Unmarshal(body, secretOpt)
		if err != nil {
			writeResponse(rw, common.NewErrorResponse(fmt.Errorf("parse body error %s", err)))
			return
		}
	}
	opt := roomApi.NewRoomInfo(name, secretOpt.Secret, secretOpt.UseSecret, tags, group, address)
	_, err = s.roomManager.CreateLocalRoom(r.Context(), opt)
	if err != nil {
		writeResponse(rw, common.NewErrorResponse(err))
		return
	}

	metric.Count("tunnel_room", map[string]string{
		"action": "create",
		"code":   "success",
	}, 1)

	joinLog.Infof("create group %s room", group)
	writeResponse(rw, common.NewSuccessResponse(opt))
}

func (s *WebSocket) JoinRoom(rw http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		joinLog.Error(fmt.Errorf("websocket upgrader error%w", err))
		return
	}
	defer conn.Close()

	id := r.URL.Query().Get("address")
	group := r.URL.Query().Get("group")
	name := r.URL.Query().Get("name")
	userId := r.URL.Query().Get("userId")
	forceCreate := r.URL.Query().Get("forceCreate")
	address, err := eventApi.NewAddressFromID(id)
	secretOpt := &RoomOptions{
		Secret:    r.URL.Query().Get("secret"),
		UseSecret: r.URL.Query().Get("useSecret") == "true",
	}
	socket := &socket{conn: conn}
	if err != nil {
		socket.writeWebsocketError(roomApi.NewRoomNotFoundError(err.Error()))
		return
	}

	connection := s.roomManager.CreateConnection()
	connection.Name = name
	connection.UserID = userId
	joinOpt := &roomApi.Info{
		BasicInfo: roomApi.BasicInfo{
			Group: group,
		},
		Address: address,
		Secret:  secretOpt.Secret,
	}

	var room roomApi.RemoteRoom
	if forceCreate == "true" {
		opt := roomApi.NewRoomInfo("", secretOpt.Secret, secretOpt.UseSecret, map[string]string{}, "", address)
		room, err = s.roomManager.ForceJoinRoom(r.Context(), connection, joinOpt, opt)
	} else {
		room, err = s.roomManager.JoinRoom(r.Context(), connection, joinOpt)
	}

	if err != nil {
		socket.writeWebsocketError(fmt.Errorf("get room user list failed, %w", err))
		return
	}

	users, err := s.roomManager.GetRoomUsers(r.Context(), joinOpt)
	if err != nil {
		socket.writeWebsocketError(fmt.Errorf("get room user list failed, %w", err))
		return
	}

	msg := roomApi.NewConnectMessage(connection, users)
	bs, err := json.Marshal(msg)
	if err != nil {
		joinLog.WithError(err).Error("send connect message marshal error")
	}

	err = socket.WriteMessage(websocket.TextMessage, bs)
	if err != nil {
		joinLog.WithError(err).Error("send connect message error")
	}

	s.serveRoom(joinOpt, connection, socket, room)
}

func (s *WebSocket) CheckRoomSecret(rw http.ResponseWriter, r *http.Request) {
	secret := r.URL.Query().Get("secret")
	if secret == "" {
		writeResponse(rw, common.NewErrorResponse(fmt.Errorf("'secret' cannot be empty")))
		return
	}

	id := r.URL.Query().Get("address")
	address, err := eventApi.NewAddressFromID(id)

	if err != nil {
		writeResponse(rw, common.NewErrorResponse(err))
		return
	}

	room, err := s.roomManager.GetRoom(r.Context(), &roomApi.Info{
		Address: address,
	})

	if err != nil {
		writeResponse(rw, common.NewErrorResponse(err))
		return
	}

	if secret == room.GetInfo().Secret {
		writeResponse(rw, common.NewSuccessResponse(nil))
	} else {
		writeResponse(rw, common.NewErrorResponse(fmt.Errorf("wrong secret")))
	}
}
