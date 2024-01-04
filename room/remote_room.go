package room

import (
	"context"
	"fmt"
	"time"

	"github.com/HuolalaTech/page-spy-api/api/event"
	"github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/metric"
	"github.com/HuolalaTech/page-spy-api/state"
	"github.com/sirupsen/logrus"
)

func NewRemoteRoom(connection *room.Connection, opt *room.Info, eventEmitter event.EventEmitter, rpcRoom room.RpcRoom) (room.RemoteRoom, error) {
	r := &remoteRoom{
		basicRoom:    newBasicRoom(),
		connection:   connection,
		opt:          opt,
		log:          log.WithField("remote_room", connection.Address.ID).WithField("local_room", opt.Address.ID),
		eventEmitter: eventEmitter,
		rpcRoom:      rpcRoom,
		messages:     make(chan *room.Message, 20),
		createdAt:    time.Now(),
		activeAt:     time.Now(),
	}
	r.log.Infof("remote room %s created", opt.Address.ID)
	return r, nil
}

type remoteRoom struct {
	*basicRoom
	log          *logrus.Entry
	connection   *room.Connection
	opt          *room.Info
	eventEmitter event.EventEmitter
	rpcRoom      room.RpcRoom
	messages     chan *room.Message
	createdAt    time.Time
	activeAt     time.Time
}

func (r *remoteRoom) GetRoomAddress() *event.Address {
	return r.rpcRoom.GetRoomAddress()
}

func (r *remoteRoom) GetInfo() *room.Info {
	return r.rpcRoom.GetInfo()
}

func (r *remoteRoom) Start(ctx context.Context) error {
	r.log.Infof("remote room %s started", r.opt.Address.ID)
	metric.Count("tunnel_remote_room", map[string]string{
		"action": "start",
		"code":   "success",
	}, 1)
	r.eventEmitter.Listen(r.connection.Address, r)
	return nil
}

func (r *remoteRoom) message(ctx context.Context, msg *room.Message) error {
	content, ok := msg.Content.(*room.MessageMessageContent)
	if !ok {
		return fmt.Errorf("message content is invalid")
	}

	if content.To == nil {
		return fmt.Errorf("unicast message's field 'to' is empty")
	}

	content.From = r.connection
	eventMsg, err := roomMessageToPackage(msg, r.connection.Address)
	if err != nil {
		return err
	}

	return r.eventEmitter.Emit(ctx, content.To.Address, eventMsg)
}

func (r *remoteRoom) ping(ctx context.Context) error {
	msg := room.NewPingMessage(*r.connection.Address)
	eventMsg, err := roomMessageToPackage(msg, r.connection.Address)
	if err != nil {
		return err
	}

	return r.eventEmitter.Emit(ctx, r.opt.Address, eventMsg)
}

func (r *remoteRoom) broadcast(ctx context.Context, msg *room.Message) error {
	content, ok := msg.Content.(*room.BroadcastMessageContent)
	if !ok {
		return fmt.Errorf("message content is invalid")
	}

	content.From = r.connection

	eventMsg, err := roomMessageToPackage(msg, r.connection.Address)
	if err != nil {
		return err
	}

	return r.eventEmitter.Emit(ctx, r.opt.Address, eventMsg)
}

func (r *remoteRoom) SendMessage(ctx context.Context, msg *room.Message) error {
	if room.NotMessageType(msg.Type) {
		return fmt.Errorf("message type %s not found", msg.Type)
	}

	r.activeAt = time.Now()
	switch msg.Type {
	case room.MessageType:
		return r.message(ctx, msg)
	case room.BroadcastType:
		return r.broadcast(ctx, msg)
	case room.PingType:
		return r.ping(ctx)
	}

	return fmt.Errorf("message type %s is not supported to be sent by normal user", msg.Type)
}

func (r *remoteRoom) OnMessage() chan *room.Message {
	return r.messages
}

func (r *remoteRoom) Close(ctx context.Context) error {
	r.log.Infof("room closed")
	err := r.close(ctx)
	if err != nil {
		return err
	}

	metric.Count("tunnel_remote_room", map[string]string{
		"action": "close",
		"code":   "close",
	}, 1)
	r.eventEmitter.RemoveListener(r.connection.Address, r)
	return nil
}

func (r *remoteRoom) ShouldRemove() bool {
	if r.StatusMachine.IsStatus(state.CloseStatus) {
		return true
	}

	now := time.Now()
	return now.Sub(r.createdAt) > 1*time.Hour || now.Sub(r.activeAt) > 20*time.Second
}

func (r *remoteRoom) Listen(ctx context.Context, msg *event.Package) {
	roomMsg, err := packageToRoomMessage(msg)
	if err != nil {
		r.log.WithError(err).Error("listen messageToRoomMessage failed")
		return
	}

	select {
	case r.messages <- roomMsg:
		if roomMsg.Type == room.CloseType {
			r.log.Infof("received close message")
			r.Close(ctx)
		}

		return
	case <-ctx.Done():
		r.log.Errorf("consume message %s timeout", msg.Content)
		return
	}
}
