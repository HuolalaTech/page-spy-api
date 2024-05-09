package room

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HuolalaTech/page-spy-api/api/event"
)

const (
	BroadcastType      = "broadcast"
	ConnectType        = "connect"
	StartType          = "start"
	CloseType          = "close"
	PingType           = "ping"
	PongType           = "pong"
	UpdateRoomInfoType = "updateRoomInfo"
	MessageType        = "message"
	JoinType           = "join"
	ErrorType          = "error"
	LeaveType          = "leave"
	UnknownType        = "unknown"
)

type RawMessage struct {
	Type      string          `json:"type"`
	CreatedAt int64           `json:"createdAt"`
	RequestId string          `json:"requestId"`
	Content   json.RawMessage `json:"content"`
}

func (rm *RawMessage) ToMessage() (*Message, error) {
	content := NewMessageContent(rm.Type)
	err := json.Unmarshal(rm.Content, content)
	if err != nil {
		return nil, fmt.Errorf("Raw message to message error %w", err)
	}

	return &Message{
		Type:      rm.Type,
		CreatedAt: rm.CreatedAt,
		RequestId: rm.RequestId,
		Content:   content,
	}, nil
}

type Message struct {
	Type      string      `json:"type"`
	CreatedAt int64       `json:"createdAt"`
	RequestId string      `json:"requestId"`
	Content   interface{} `json:"content"`
}

func (m *Message) GetPong() *Message {
	return &Message{
		Type:      PongType,
		CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
		RequestId: m.RequestId,
		Content:   map[string]string{},
	}
}

func (m *Message) ToString() string {
	bs, err := json.Marshal(m)
	if err != nil {
		return ""
	}

	return string(bs)
}

func IsPublicMessageType(messageType string) bool {
	switch messageType {
	case BroadcastType, UpdateRoomInfoType, MessageType, PingType:
		return true
	}

	return false
}

func NotMessageType(messageType string) bool {
	switch messageType {
	case BroadcastType, MessageType:
		return false
	case PingType:
		return false
	case CloseType, StartType:
		return false
	case ErrorType:
		return false
	case JoinType, LeaveType:
		return false
	case UpdateRoomInfoType:
		return false
	case UnknownType:
		return true
	}

	return true
}

func NewMessageContent(messageType string) interface{} {
	var unknownContent interface{}
	switch messageType {
	case MessageType:
		return &MessageMessageContent{}
	case BroadcastType:
		return &BroadcastMessageContent{}
	case PingType:
		return &PingContent{}
	case UpdateRoomInfoType:
		return &UpdateRoomInfoContent{}
	case CloseType, StartType:
		return &StartOrCloseMessageContent{}
	case ErrorType:
		return &ErrorMessageContent{}
	case JoinType, LeaveType:
		return &JoinOrLeaveMessageContent{}
	case UnknownType:
		return &unknownContent
	}

	return &unknownContent
}

type UpdateRoomInfoContent struct {
	Info *Info `json:"info"`
}

type PingContent struct {
	From event.Address `json:"from"`
}

type ErrorMessageContent struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func NewPingMessage(from event.Address) *Message {
	return &Message{
		Type:      PingType,
		CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
		Content:   &PingContent{From: from},
	}
}

type StartOrCloseMessageContent struct {
	RoomAddress event.Address `json:"roomAddress"`
	Reason      string        `json:"reason"`
}

func NewStartMessage(roomAddress event.Address) *Message {
	return &Message{
		Type:      StartType,
		CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
		Content:   &StartOrCloseMessageContent{RoomAddress: roomAddress},
	}
}

func NewCloseMessage(roomAddress event.Address, reason string) *Message {
	return &Message{
		Type:      CloseType,
		CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
		Content:   &StartOrCloseMessageContent{RoomAddress: roomAddress, Reason: reason},
	}
}

type ConnectMessageContent struct {
	SelfConnection  *Connection   `json:"selfConnection"`
	RoomConnections []*Connection `json:"roomConnections"`
}

func NewConnectMessage(selfConnection *Connection, roomConnections []*Connection) *Message {
	return &Message{
		Type:      ConnectType,
		CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
		Content:   &ConnectMessageContent{SelfConnection: selfConnection, RoomConnections: roomConnections},
	}
}

type JoinOrLeaveMessageContent struct {
	Connection *Connection `json:"connection"`
}

func NewLeaveMessage(connection *Connection) *Message {
	return &Message{
		Type:      LeaveType,
		CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
		Content:   &JoinOrLeaveMessageContent{Connection: connection},
	}
}

func NewJoinMessage(connection *Connection) *Message {
	return &Message{
		Type:      JoinType,
		CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
		Content:   &JoinOrLeaveMessageContent{Connection: connection},
	}
}

type MessageMessageContent struct {
	Data json.RawMessage `json:"data"`
	From *Connection     `json:"from"`
	To   *Connection     `json:"to"`
}

type BroadcastMessageContent struct {
	Data        json.RawMessage `json:"data"`
	From        *Connection     `json:"from"`
	IncludeSelf bool            `json:"includeSelf"`
}

func NewBroadcastMessage(data json.RawMessage, from *Connection) *Message {
	return &Message{
		Type:      BroadcastType,
		CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
		Content: &BroadcastMessageContent{
			Data: data,
			From: from,
		},
	}
}

func NewSendMessage(data json.RawMessage, from *Connection, to *Connection) *Message {
	return &Message{
		Type:      MessageType,
		CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
		Content: &MessageMessageContent{
			Data: data,
			From: from,
			To:   to,
		},
	}
}

type Connection struct {
	Address   *event.Address `json:"address"`
	CreatedAt time.Time      `json:"createdAt"`
	UserID    string         `json:"userId"`
	Name      string         `json:"name"`
}

type BasicInfo struct {
	Name  string            `json:"name"`
	Group string            `json:"group"`
	Tags  map[string]string `json:"tags"`
}
type Info struct {
	BasicInfo
	Address     *event.Address `json:"address"`
	Secret      string         `json:"secret"`
	UseSecret   bool           `json:"useSecret"`
	CreatedAt   time.Time      `json:"createdAt"`
	ActiveAt    time.Time      `json:"activeAt"`
	Connections []*Connection  `json:"connections"`
}

func (i *Info) Update(info *Info) {
	if info.Name != "" {
		i.Name = info.Name
	}

	if info.Group != "" {
		i.Group = info.Group
	}

	if len(info.Tags) > 0 {
		i.Tags = info.Tags
	}
}

func NewRoomInfo(name string, secret string, useSecret bool, tags map[string]string, group string, address *event.Address) *Info {
	return &Info{
		BasicInfo: BasicInfo{
			Group: group,
			Tags:  tags,
			Name:  name,
		},
		Address:     address,
		Secret:      secret,
		UseSecret:   useSecret,
		Connections: make([]*Connection, 0),
		CreatedAt:   time.Now(),
		ActiveAt:    time.Now(),
	}
}

type RpcRoom interface {
	GetRoomAddress() *event.Address
	GetInfo() *Info
	UpdateInfo(info *Info)
}

type ManagerRoom interface {
	RpcRoom
	ShouldRemove() bool
	Close(ctx context.Context) error
	IsClose() bool
}

type RemoteRoom interface {
	ManagerRoom
	Start(ctx context.Context) error
	SendMessage(ctx context.Context, msg *Message) error
	OnMessage() chan *Message
	Done() chan struct{}
}

type Room interface {
	RemoteRoom
	Ping()
	GetRoomUsers() []*Connection
	Join(ctx context.Context, connection *Connection, opt *Info) error
	Leave(ctx context.Context, connection *Connection, opt *Info) error
}
