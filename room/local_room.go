package room

import (
	"context"
	"encoding/json"
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

func NewLocalRoom(opt *room.Info, event event.EventEmitter, addressManager *rpc.AddressManager) (room.Room, error) {
	if opt == nil || opt.Address == nil {
		return nil, fmt.Errorf("room info or address is nil")
	}

	if opt.UseSecret && opt.Secret == "" {
		return nil, fmt.Errorf("room %s use secret but secret is empty", opt.Address.ID)
	}

	opt.Connections = make([]*room.Connection, 0)
	opt.CreatedAt = time.Now()
	opt.ActiveAt = time.Now()
	info := cloneRoomInfo(opt)

	logger := log.WithField("room", info.Address.ID)
	logger.Infof("local room created")

	return &localRoom{
		basicRoom:   newBasicRoom(),
		closeCode:   "unknown",
		closeReason: "unknown",
		log:         logger,
		Info:        info,
		event:       event,
		messages:    make(chan *room.Message, 2000),
	}, nil
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
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()
	return cloneAddress(r.Info.Address)
}

func (r *localRoom) GetRoomUsers() []*room.Connection {
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()
	return cloneConnections(r.Info.Connections)
}

func (r *localRoom) GetGroup() string {
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()
	return r.Info.Group
}

func (r *localRoom) GetInfo() *room.Info {
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()
	return cloneRoomInfo(r.Info)
}

func (r *localRoom) UpdateInfo(info *room.Info) {
	if info == nil {
		return
	}

	r.rwLock.Lock()
	defer r.rwLock.Unlock()
	r.Info.Update(cloneRoomInfo(info))
}

func (r *localRoom) GetTags() map[string]string {
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()
	return cloneTags(r.Info.Tags)
}

func (r *localRoom) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Info *room.Info `json:"Info"`
	}{
		Info: r.GetInfo(),
	})
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
				err := r.SendMessage(context.Background(), msg)
				if err != nil {
					r.log.WithError(err).Errorf("local room broadcast messages failed, %s", err)
				}
			case <-r.Done():
				return
			}

		}
	}()
	address := r.GetRoomAddress()
	r.event.Listen(address, r)
	r.SendMessageWithTimeout(room.NewStartMessage(*address), 5*time.Second)
	return nil
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

func (r *localRoom) touchActiveAt() {
	r.rwLock.Lock()
	r.Info.ActiveAt = time.Now()
	r.rwLock.Unlock()
}

func (r *localRoom) getMessageSnapshot() (*event.Address, []*room.Connection) {
	r.rwLock.Lock()
	defer r.rwLock.Unlock()
	r.Info.ActiveAt = time.Now()
	return cloneAddress(r.Info.Address), cloneConnections(r.Info.Connections)
}

func (r *localRoom) Join(ctx context.Context, connection *room.Connection, opt *room.Info) error {
	if opt == nil || opt.Address == nil || connection == nil || connection.Address == nil {
		return fmt.Errorf("join room info or connection is invalid")
	}

	r.rwLock.Lock()
	if !r.Info.Address.Equal(opt.Address) {
		r.rwLock.Unlock()
		return fmt.Errorf("connection %s join room %s failed", connection.Address.ID, opt.Address.ID)
	}

	if r.Info.UseSecret && r.Info.Secret != opt.Secret {
		r.rwLock.Unlock()
		return fmt.Errorf("join failed, password from connection %s of room %s is invalid", connection.Address.ID, opt.Address.ID)
	}
	r.Info.Connections = append(r.Info.Connections, cloneConnection(connection))
	r.Info.ActiveAt = time.Now()
	r.rwLock.Unlock()

	r.log.Infof("connection %s joined room", connection.Address.ID)
	r.SendMessageWithTimeout(room.NewJoinMessage(connection), 5*time.Second)
	r.SetStatus(state.RunningStatus)
	return nil
}

func (r *localRoom) Leave(ctx context.Context, connection *room.Connection, opt *room.Info) error {
	if opt == nil || opt.Address == nil || connection == nil || connection.Address == nil {
		return fmt.Errorf("leave room info or connection is invalid")
	}

	address := r.GetRoomAddress()
	if !address.Equal(opt.Address) {
		return fmt.Errorf("connection %s leave room %s failed", connection.Address.ID, opt.Address.ID)
	}

	r.log.Infof("connection %s left room %s", connection.Address.ID, opt.Address.ID)
	r.removeConnectionWithLock(connection)
	r.SendMessageWithTimeout(room.NewLeaveMessage(connection), 5*time.Second)
	return nil
}

func (r *localRoom) Ping() {
	r.touchActiveAt()
}

func (r *localRoom) pingMessage() error {
	r.Ping()
	return nil
}

func (r *localRoom) otherMessage(ctx context.Context, msg *room.Message) error {
	address, connections := r.getMessageSnapshot()
	eventMsg, err := roomMessageToPackage(msg, address)
	if err != nil {
		return err
	}

	for _, c := range connections {
		e := r.event.Emit(ctx, c.Address, eventMsg)
		if e != nil {
			r.log.WithError(e).Errorf("emit connection %s message failed", c.Address.ID)
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

	if content.From == nil || content.From.Address == nil {
		return fmt.Errorf("broadcast message's field 'from.address' is empty")
	}

	address, connections := r.getMessageSnapshot()
	eventMsg, err := roomMessageToPackage(msg, address)
	if err != nil {
		return err
	}

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

	if content.To == nil || content.To.Address == nil {
		return fmt.Errorf("unicast message's field 'to.address' is empty")
	}

	address, connections := r.getMessageSnapshot()
	eventMsg, err := roomMessageToPackage(msg, address)
	if err != nil {
		return err
	}

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

	r.touchActiveAt()
	switch msg.Type {
	case room.MessageType:
		return r.messageMessage(ctx, msg)
	case room.BroadcastType:
		return r.broadcastMessage(ctx, msg)
	case room.PingType:
		return r.pingMessage()
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

func (r *localRoom) Close(ctx context.Context, closeCode string) error {
	if !r.close() {
		return nil
	}

	r.rwLock.RLock()
	address := cloneAddress(r.Info.Address)
	reason := r.closeReason
	code := r.closeCode
	r.rwLock.RUnlock()

	metric.Count("tunnel_local_room", map[string]string{
		"action": "close",
		"code":   code,
	}, 1)

	r.event.RemoveListener(address, r)
	r.log.Infof("room closed, %s", reason)
	r.SendMessageWithTimeout(room.NewCloseMessage(*address, reason), 5*time.Second)
	return nil
}

func (r *localRoom) ShouldRemove() (string, bool) {
	r.rwLock.Lock()
	defer r.rwLock.Unlock()

	if r.StatusMachine.IsStatus(state.CloseStatus) {
		return r.closeCode, true
	}

	now := time.Now()
	isEmpty := len(r.Info.Connections) <= 0
	noUseInitRoom := r.IsStatus(state.InitStatus) && isEmpty && now.Sub(r.Info.CreatedAt) > 1*time.Minute
	noUserRoom := r.IsStatus(state.RunningStatus) && isEmpty && now.Sub(r.Info.ActiveAt) > 1*time.Minute
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

	return r.closeCode, noUseInitRoom || noUserRoom || noUseRoom || maxTimeRoom
}

func (r *localRoom) Listen(ctx context.Context, pkg *event.Package) {
	roomMsg, err := packageToRoomMessage(pkg)
	if err != nil {
		r.log.WithError(err).Error("listen message failed")
		return
	}
	start := time.Now()
	status := "success"
	defer func() {
		metric.Time("page_spy_local_room_emit", map[string]string{
			"status": status,
		}, float64(time.Since(start).Milliseconds()))
	}()

	select {
	case r.messages <- roomMsg:
		return
	case <-ctx.Done():
		status = "timeout"
		r.log.Errorf("listen message %s timeout", pkg.Content)
		return
	}
}
