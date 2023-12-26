package socket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"
	"sync"

	eventApi "github.com/HuolalaTech/page-spy-api/api/event"
	roomApi "github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/logger"
	"github.com/HuolalaTech/page-spy-api/metric"
	"github.com/HuolalaTech/page-spy-api/room"
	"github.com/gorilla/websocket"
)

var joinLog = logger.Log().WithField("module", "join")

type Response struct {
	Code    string      `json:"code"`
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
	Message string      `json:"message"`
}

func NewErrorResponse(err error) *Response {
	return &Response{
		Code:    "error",
		Success: false,
		Message: err.Error(),
	}
}

func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Code:    "success",
		Data:    data,
		Success: true,
	}
}

func writeResponse(w http.ResponseWriter, res *Response) {
	if res.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	bs, err := json.Marshal(res)
	if err != nil {
		joinLog.WithError(err).Error("write  message error")
	}

	_, err = w.Write(bs)
	if err != nil {
		joinLog.WithError(err).Error("write  message error")
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

func readClientMessage(ctx context.Context, socket *socket, room roomApi.RemoteRoom) error {
	if room.IsClose() {
		return roomApi.NewRoomCloseError("room %s is already close", room.GetRoomAddress().ID)
	}

	rawMsg := &roomApi.RawMessage{}
	err := socket.ReadJSON(rawMsg)
	if err != nil {
		_, ok := err.(*websocket.CloseError)
		if ok {
			return roomApi.NewRoomCloseError("read message websocket error %s", err.Error())
		}

		socket.writeWebsocketError(roomApi.NewRoomCloseError("读取消息解析错误 %s", err.Error()))
		return nil
	}
	msg, err := rawMsg.ToMessage()

	if err != nil {
		socket.writeWebsocketError(roomApi.NewRoomCloseError("消息转换格式错误%s", err.Error()))
		return nil
	}

	if !roomApi.IsPublicMessageType(msg.Type) {
		socket.writeWebsocketError(roomApi.NewRoomCloseError("前端不能发送消息类型 %s", msg.Type))
		return nil
	}
	log.Debugf("socket 接受信息 %s", msg.Type)
	err = room.SendMessage(ctx, msg)
	if err != nil {
		socket.writeWebsocketError(err)
		return nil
	}

	socket.WriteDataIgnoreError(msg.GetReply())
	return nil
}

func onRoomMessage(ctx context.Context, socket *socket, room roomApi.RemoteRoom) error {
	select {
	case msg := <-room.OnMessage():
		socket.WriteDataIgnoreError(msg)
	case <-room.Done():
		return roomApi.NewRoomCloseError("room %s leave", room.GetRoomAddress().ID)
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
			err := readClientMessage(cancelCtx, socket, room)
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
	tags := getTags(r.URL.Query())
	rooms, err := s.roomManager.ListRooms(r.Context(), tags)
	if err != nil {
		writeResponse(rw, NewErrorResponse(err))
		return
	}

	writeResponse(rw, NewSuccessResponse(rooms))
}

type CreateRoomParams struct {
	Group    string            `json:"group"`
	Name     string            `json:"name"`
	Password string            `json:"password"`
	Tags     map[string]string `json:"tags"`
}

func getTags(query url.Values) map[string]string {
	tags := make(map[string]string, len(query))
	for k, v := range query {
		if len(v) > 0 {
			value := v[0]
			tags[k] = value
		}
	}

	return tags
}

func (s *WebSocket) CreateRoom(rw http.ResponseWriter, r *http.Request) {
	address := s.roomManager.AddressManager.GeneratorRoomAddress()
	name := r.URL.Query().Get("name")
	group := r.URL.Query().Get("group")
	tags := getTags(r.URL.Query())
	if name == "" || group == "" {
		writeResponse(rw, NewErrorResponse(errors.New("name 或者 group 参数缺失")))
		return
	}

	opt := roomApi.NewRoomInfo(name, address.ID, group, address)
	opt.Tags = tags
	_, err := s.roomManager.CreateRoom(r.Context(), opt)
	if err != nil {
		writeResponse(rw, NewErrorResponse(err))
		return
	}

	metric.Count("tunnel_room", map[string]string{
		"action": "create",
		"code":   "success",
	}, 1)

	joinLog.Infof("create group %s room", group)
	writeResponse(rw, NewSuccessResponse(opt))
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
	address, err := eventApi.NewAddressFromID(id)
	socket := &socket{conn: conn}
	if err != nil {
		socket.writeWebsocketError(roomApi.NewRoomNotFoundError(err.Error()))
		return
	}

	connection := s.roomManager.CreateConnection()
	connection.Name = name
	connection.UserID = userId
	opt := &roomApi.Info{
		Address:  address,
		Password: address.ID,
		Group:    group,
	}

	room, err := s.roomManager.JoinRoom(r.Context(), connection, opt)
	if err != nil {
		socket.writeWebsocketError(fmt.Errorf("加入房间错误%w", err))
		return
	}

	users, err := s.roomManager.GetRoomUsers(r.Context(), opt)
	if err != nil {
		socket.writeWebsocketError(fmt.Errorf("获取房间用户列表%w", err))
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

	s.serveRoom(opt, connection, socket, room)
}
