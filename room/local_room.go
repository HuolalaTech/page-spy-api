package room

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/HuolalaTech/page-spy-api/api/event"
	"github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/metric"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/HuolalaTech/page-spy-api/state"
	"github.com/sirupsen/logrus"
)

func NewLocalRoom(opt *room.Info, event event.EventEmitter, addressManager *rpc.AddressManager) room.Room {
	logger := log.WithField("room", opt.Address.ID)
	logger.Infof("local room created")
	opt.Connections = make([]*room.Connection, 0)
	return &localRoom{
		basicRoom: newBasicRoom(),
		log:       logger,
		Info:      opt,
		event:     event,
		messages:  make(chan *room.Message, 10),
	}
}

type localRoom struct {
	*basicRoom
	closeReason string
	closeCode   string
	log         *logrus.Entry
	rwLock      sync.RWMutex
	Info        *room.Info
	event       event.EventEmitter
	messages    chan *room.Message
}

func (r *localRoom) GetRoomAddress() *event.Address {
	return r.Info.Address
}

func (r *localRoom) GetRoomUsers() []*room.Connection {
	return r.Info.Connections
}

func (r *localRoom) GetGroup() string {
	return r.Info.Group
}

func (r *localRoom) GetInfo() *room.Info {
	return r.Info
}

func (r *localRoom) GetTags() map[string]string {
	return r.Info.Tags
}

func (r *localRoom) Start(ctx context.Context) error {
	r.log.Infof("room started")
	metric.Count("tunnel_local_room", map[string]string{
		"action": "start",
		"code":   "success",
	}, 1)
	go func() {
		for {
			select {
			case msg := <-r.OnMessage():
				if msg.Type == room.PingType {
					r.Ping()
				} else {
					err := r.SendMessage(context.Background(), msg)
					if err != nil {
						r.log.WithError(err).Errorf("local room broadcast messages failed, %s", err)
					}
				}
			case <-r.Done():
				return
			}

		}
	}()
	r.event.Listen(r.Info.Address, r)
	r.SendMessageWithTimeout(room.NewStartMessage(*r.Info.Address), 5*time.Second)
	return nil
}

func (r *localRoom) addConnectionWithLock(connection *room.Connection) {
	r.rwLock.Lock()
	defer r.rwLock.Unlock()
	r.Info.Connections = append(r.Info.Connections, connection)
}

func (r *localRoom) removeConnectionWithLock(connection *room.Connection) {
	r.rwLock.Lock()
	defer r.rwLock.Unlock()
	newConnections := make([]*room.Connection, 0)
	for _, c := range r.Info.Connections {
		if !c.Address.Equal(connection.Address) {
			newConnections = append(newConnections, c)
		}
	}

	r.Info.Connections = newConnections
}

func (r *localRoom) getConnectionsWithLock() []*room.Connection {
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()
	return r.Info.Connections
}

func (r *localRoom) Join(ctx context.Context, connection *room.Connection, opt *room.Info) error {
	if opt == nil {
		return nil
	}

	if !r.Info.Address.Equal(opt.Address) {
		return fmt.Errorf("connection %s join room %s failed", connection.Address.ID, opt.Address.ID)
	}

	if r.Info.Password != opt.Password {
		return fmt.Errorf("join failed, password from connection %s of room %s is invalid, correct password is %s", connection.Address.ID, opt.Address.ID, opt.Password)
	}

	r.log.Infof("connection %s joined room", connection.Address.ID)
	r.addConnectionWithLock(connection)
	r.SendMessageWithTimeout(room.NewJoinMessage(connection), 5*time.Second)
	r.SetStatus(state.RunningStatus)
	return nil
}

func (r *localRoom) Leave(ctx context.Context, connection *room.Connection, opt *room.Info) error {
	if opt == nil {
		return nil
	}

	if !r.Info.Address.Equal(opt.Address) {
		return fmt.Errorf("connection %s leave room %s failed", connection.Address.ID, opt.Address.ID)
	}

	if r.Info.Password != opt.Password {
		return fmt.Errorf("leave failed, password from connection %s of room %s is invalid, correct password is %s", connection.Address.ID, opt.Address.ID, opt.Password)
	}

	r.log.Infof("connection %s left room %s", connection.Address.ID, opt.Address.ID)
	r.removeConnectionWithLock(connection)
	r.SendMessageWithTimeout(room.NewLeaveMessage(connection), 5*time.Second)
	return nil
}

func (r *localRoom) Ping() {
	r.Info.ActiveAt = time.Now()
}

func (r *localRoom) pingMessage(ctx context.Context) error {
	r.Ping()
	return nil
}

func (r *localRoom) otherMessage(ctx context.Context, msg *room.Message) error {
	connections := r.getConnectionsWithLock()
	eventMsg, err := roomMessageToPackage(msg, r.Info.Address)
	if err != nil {
		return err
	}

	r.Info.ActiveAt = time.Now()
	for _, c := range connections {
		e := r.event.Emit(ctx, c.Address, eventMsg)
		if e != nil {
			r.log.WithError(e).Errorf("emit connection %s message failed, %s", c.Address.ID, e)
			err = e
		}
	}

	return err
}

func (r *localRoom) broadcastMessage(ctx context.Context, msg *room.Message) error {
	content, ok := msg.Content.(*room.BroadcastMessageContent)
	if !ok {
		return fmt.Errorf("message format is invalid")
	}

	connections := r.getConnectionsWithLock()
	eventMsg, err := roomMessageToPackage(msg, r.Info.Address)
	if err != nil {
		return err
	}

	r.Info.ActiveAt = time.Now()
	for _, c := range connections {
		if !(c.Address.Equal(content.From.Address) && !content.IncludeSelf) {
			e := r.event.Emit(ctx, c.Address, eventMsg)
			if e != nil {
				r.log.WithError(e).Errorf("emit connection %s message failed, %s", c.Address.ID, e.Error())
				err = e
			}
		}
	}

	return err
}

func (r *localRoom) messageMessage(ctx context.Context, msg *room.Message) error {
	content, ok := msg.Content.(*room.MessageMessageContent)
	if !ok {
		return fmt.Errorf("message format is invalid")
	}

	if content.To == nil {
		return fmt.Errorf("unicast message's field 'to' is empty")
	}

	connections := r.getConnectionsWithLock()
	eventMsg, err := roomMessageToPackage(msg, r.Info.Address)
	if err != nil {
		return err
	}

	r.Info.ActiveAt = time.Now()
	for _, c := range connections {
		if c.Address.Equal(content.To.Address) {
			e := r.event.Emit(ctx, c.Address, eventMsg)
			if e != nil {
				r.log.WithError(e).Errorf("emit connection %s message failed, %s", c.Address.ID, e.Error())
				err = e
			}
		}
	}

	return err
}

func (r *localRoom) SendMessage(ctx context.Context, msg *room.Message) error {
	if room.NotMessageType(msg.Type) {
		return fmt.Errorf("message type %s not found", msg.Type)
	}

	r.Info.ActiveAt = time.Now()
	if msg.Type == room.PingType {
		return r.pingMessage(ctx)
	}

	switch msg.Type {
	case room.MessageType:
		return r.messageMessage(ctx, msg)
	case room.BroadcastType:
		return r.broadcastMessage(ctx, msg)
	case room.PingType:
		return r.pingMessage(ctx)
	}

	if !room.NotMessageType(msg.Type) {
		return r.otherMessage(ctx, msg)
	}

	return fmt.Errorf("message type %s is not supported to be sent by normal user", msg.Type)
}

func (r *localRoom) SendMessageWithTimeout(msg *room.Message, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := r.SendMessage(ctx, msg)
	if err != nil {
		r.log.Error(err)
	}
}

func (r *localRoom) OnMessage() chan *room.Message {
	return r.messages
}

func (r *localRoom) Close(ctx context.Context) error {
	if r.StatusMachine.IsStatus(state.CloseStatus) {
		return nil
	}
	metric.Count("tunnel_local_room", map[string]string{
		"action": "close",
		"code":   r.closeCode,
	}, 1)
	err := r.close(ctx)
	if err != nil {
		return err
	}

	r.event.RemoveListener(r.Info.Address, r)
	r.log.Infof("room closed, %s", r.closeReason)
	r.SendMessageWithTimeout(room.NewCloseMessage(*r.Info.Address, r.closeReason), 5*time.Second)
	return nil
}

func (r *localRoom) ShouldRemove() bool {
	if r.StatusMachine.IsStatus(state.CloseStatus) {
		return true
	}

	now := time.Now()
	noUseInitRoom := r.IsStatus(state.InitStatus) && r.isEmpty() && now.Sub(r.Info.CreatedAt) > 1*time.Minute
	noUserRoom := r.IsStatus(state.RunningStatus) && r.isEmpty() && now.Sub(r.Info.ActiveAt) > 1*time.Minute
	noUseRoom := r.IsStatus(state.RunningStatus) && now.Sub(r.Info.ActiveAt) > 5*time.Minute
	maxTimeRoom := now.Sub(r.Info.CreatedAt) > 1*time.Hour
	switch true {
	case noUseInitRoom:
		r.closeReason = "no user connection for more than 1 minute after room setup"
		r.closeCode = "noUseInitRoom"
	case noUserRoom:
		r.closeReason = "all the user of room left over 1 minutes"
		r.closeCode = "noUserRoom"
	case noUseRoom:
		r.closeReason = "room idle over 5 minutes"
		r.closeCode = "noUseRoom"
	case maxTimeRoom:
		r.closeReason = "room exceeded the maximum time 1 hour"
		r.closeCode = "maxTimeRoom"
	}

	return noUseInitRoom || noUserRoom || noUseRoom || maxTimeRoom
}

func (r *localRoom) isEmpty() bool {
	connections := r.getConnectionsWithLock()
	return len(connections) <= 0
}

func (r *localRoom) Listen(ctx context.Context, pkg *event.Package) {
	roomMsg, err := packageToRoomMessage(pkg)
	if err != nil {
		r.log.WithError(err).Error("listen message failed")
		return
	}

	select {
	case r.messages <- roomMsg:
		return
	case <-ctx.Done():
		r.log.Errorf("listen message %s timeout", pkg.Content)
		return
	}
}
