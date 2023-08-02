package room

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/HuolalaTech/page-spy-api/api/event"
	"github.com/HuolalaTech/page-spy-api/api/room"
	roomApi "github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/logger"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/sirupsen/logrus"
)

func NewLocalRoomManager(event event.EventEmitter, addressManager *rpc.AddressManager, maxRoomSize int64) *LocalRoomManager {
	return &LocalRoomManager{
		BasicManager:   *NewBasicManager(),
		event:          event,
		log:            logger.Log().WithField("module", "LocalRoomManager"),
		maxRoomSize:    maxRoomSize,
		AddressManager: addressManager,
	}
}

type LocalRoomManager struct {
	BasicManager
	AddressManager *rpc.AddressManager
	event          event.EventEmitter
	log            *logrus.Entry
	maxRoomSize    int64
}

func (r *LocalRoomManager) Start() {
	r.start()
	r.log.Info("local room manager start")
}

func (r *LocalRoomManager) CreateConnection() (*room.Connection, error) {
	address := r.AddressManager.GeneratorConnectionAddress()
	return &room.Connection{
		Address:   address,
		CreatedAt: time.Now(),
	}, nil
}

func (r *LocalRoomManager) GetRoomsByGroup(tags map[string]string) []room.Room {
	rs := r.getRoomsByTags(tags)
	rooms := make([]room.Room, 0, len(rs))
	for _, r := range rs {
		rooms = append(rooms, r.(*localRoom))
	}

	return rooms
}

func (r *LocalRoomManager) GetRooms() []room.Room {
	rs := r.getRooms()
	rooms := make([]room.Room, 0, len(rs))
	for _, r := range rs {
		rooms = append(rooms, r.(*localRoom))
	}

	return rooms
}

func (r *LocalRoomManager) isFull() bool {
	rooms := r.GetRooms()
	return len(rooms) >= int(r.maxRoomSize)
}

func (r *LocalRoomManager) CreateRoom(ctx context.Context, info *room.Info) (room.Room, error) {
	if r.isFull() {
		return nil, errors.New("房间已经超过限制，不可以再创建房间")
	}

	room := NewLocalRoom(info, r.event, r.AddressManager)
	err := room.Start(ctx)
	if err != nil {
		return nil, err
	}

	r.addRoom(room)
	return room, nil
}

func (r *LocalRoomManager) GetRoom(ctx context.Context, opt *room.Info) (room.Room, error) {
	room, exist := r.getLocalRoom(opt)
	if !exist {
		return nil, roomApi.NewRoomNotFoundError("获取房间不存在")
	}

	return room, nil
}

func (r *LocalRoomManager) RemoveRoom(ctx context.Context, opt *room.Info) error {
	room, exist := r.getRoom(opt)
	if !exist {
		return nil
	}

	r.removeRoom(room)
	return room.Close(ctx)
}

func (r *LocalRoomManager) getLocalRoom(opt *room.Info) (*localRoom, bool) {
	room, exist := r.getRoom(opt)
	if !exist {
		return nil, exist
	}

	return room.(*localRoom), true
}

func (r *LocalRoomManager) JoinRoom(ctx context.Context, opt *room.Info, connection *room.Connection) error {
	room, exist := r.getLocalRoom(opt)
	if !exist {
		return roomApi.NewRoomNotFoundError(fmt.Sprintf("加入房间%s不存在", opt.Address.ID))
	}

	if room.IsClose() {
		return roomApi.NewRoomNotFoundError(fmt.Sprintf("加入房间%s已经关闭", opt.Address.ID))
	}

	err := room.Join(ctx, connection, opt)
	if err != nil {
		return err
	}

	return nil
}

func (r *LocalRoomManager) LeaveRoom(ctx context.Context, opt *room.Info, connection *room.Connection) error {
	room, exist := r.getLocalRoom(opt)
	if !exist {
		return roomApi.NewRoomNotFoundError(fmt.Sprintf("离开房间%s不存在", opt.Address.ID))
	}

	if room.IsClose() {
		return nil
	}

	err := room.Leave(ctx, connection, opt)
	if err != nil {
		r.log.WithError(err).Errorf("room manager leave room %s error", opt.Address.ID)
	}

	if room.ShouldRemove() {
		r.log.WithError(err).Errorf("room manager close room %s ", opt.Address.ID)
		r.removeRoom(room)
		return room.Close(ctx)
	}

	return nil
}
