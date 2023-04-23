package socket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

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

func writeWebsocketError(conn *websocket.Conn, errRes error) {
	message := NewErrorMessage(errRes)
	err := conn.WriteJSON(message)
	if err != nil {
		joinLog.WithError(err).Error("write websocket  message error")
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func readClientMessage(ctx context.Context, conn *websocket.Conn, room roomApi.RemoteRoom) error {
	if room.IsClose() {
		return roomApi.NewRoomCloseError("room %s is already close", room.GetRoomAddress().ID)
	}

	rawMsg := &roomApi.RawMessage{}
	err := conn.ReadJSON(rawMsg)
	if err != nil {
		_, ok := err.(*websocket.CloseError)
		if ok {
			return roomApi.NewRoomCloseError("read message websocket error %s", err.Error())
		}

		writeWebsocketError(conn, roomApi.NewRoomCloseError("读取消息解析错误 %s", err.Error()))
		return nil
	}
	msg, err := rawMsg.ToMessage()

	if err != nil {
		writeWebsocketError(conn, roomApi.NewRoomCloseError("消息转换格式错误%s", err.Error()))
		return nil
	}

	if !roomApi.IsPublicMessageType(msg.Type) {
		writeWebsocketError(conn, roomApi.NewRoomCloseError("前端不能发送消息类型 %s", msg.Type))
		return nil
	}

	err = room.SendMessage(ctx, msg)
	if err != nil {
		writeWebsocketError(conn, err)
		return nil
	}

	return nil
}

func onRoomMessage(ctx context.Context, conn *websocket.Conn, room roomApi.RemoteRoom) error {
	select {
	case msg := <-room.OnMessage():
		bs, err := json.Marshal(msg)
		if err != nil {
			return roomApi.NewMessageContentError("send message marshal error %s", err.Error())
		}

		err = conn.WriteMessage(websocket.TextMessage, bs)
		if err != nil {
			joinLog.WithError(err).Error("send message write message")
			writeWebsocketError(conn, roomApi.NewNetWorkTimeoutError(err.Error()))
			return nil
		}

	case <-room.Done():
		return roomApi.NewRoomCloseError("room %s leave", room.GetRoomAddress().ID)
	case <-ctx.Done():
		return roomApi.NewNetWorkTimeoutError("room %s context cancel", room.GetRoomAddress().ID)
	}

	return nil
}

func (s *WebSocket) serveRoom(opt *roomApi.Info, connection *roomApi.Connection, conn *websocket.Conn, room roomApi.RemoteRoom) {
	retCode := "success"
	close := func() {
		err := s.roomManager.LeaveRoom(context.Background(), opt, connection)
		if err != nil {
			joinLog.Errorf("serveRoom %s close %w code %s", opt.Address.ID, err, retCode)
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

	conn.SetCloseHandler(func(code int, text string) error {
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
				err := onRoomMessage(cancelCtx, conn, room)
				if err != nil {
					writeCode = "write_message_close"
					writeWebsocketError(conn, err)
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
			err := readClientMessage(cancelCtx, conn, room)
			if err != nil {
				retCode = "read_message_close"
				writeWebsocketError(conn, err)
				joinLog.WithField("connection", connection.Address.ID).Info(err)
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
	group := r.URL.Query().Get("group")
	rooms, err := s.roomManager.ListRooms(r.Context(), group)
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

func (s *WebSocket) CreateRoom(rw http.ResponseWriter, r *http.Request) {
	address := s.roomManager.AddressManager.GeneratorRoomAddress()
	name := r.URL.Query().Get("name")
	group := r.URL.Query().Get("group")
	if name == "" || group == "" {
		writeResponse(rw, NewErrorResponse(errors.New("name 或者 group 参数缺失")))
		return
	}

	opt := roomApi.NewRoomInfo(name, address.ID, group, address)
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
	if err != nil {
		writeWebsocketError(conn, roomApi.NewRoomNotFoundError(err.Error()))
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
		writeWebsocketError(conn, fmt.Errorf("加入房间错误%w", err))
		return
	}

	users, err := s.roomManager.GetRoomUsers(r.Context(), opt)
	if err != nil {
		writeWebsocketError(conn, fmt.Errorf("获取房间用户列表%w", err))
		return
	}

	msg := roomApi.NewConnectMessage(connection, users)
	bs, err := json.Marshal(msg)
	if err != nil {
		joinLog.WithError(err).Error("send connect message marshal error")
	}

	err = conn.WriteMessage(websocket.TextMessage, bs)
	if err != nil {
		joinLog.WithError(err).Error("send connect message error")
	}

	s.serveRoom(opt, connection, conn, room)
}
