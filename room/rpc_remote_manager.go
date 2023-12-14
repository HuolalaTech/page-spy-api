package room

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/HuolalaTech/page-spy-api/api/event"
	"github.com/HuolalaTech/page-spy-api/api/room"
	localRpc "github.com/HuolalaTech/page-spy-api/rpc"
)

func NewRemoteRpcRoomManager(addressManager *localRpc.AddressManager,
	rpcManager *localRpc.RpcManager,
	event event.EventEmitter,
	localRoomManager *LocalRoomManager) *RemoteRpcRoomManager {

	return &RemoteRpcRoomManager{
		BasicManager:     *NewBasicManager(),
		AddressManager:   addressManager,
		rpcManager:       rpcManager,
		event:            event,
		localRoomManager: localRoomManager,
	}
}

type RemoteRpcRoomManager struct {
	BasicManager
	AddressManager   *localRpc.AddressManager
	rpcManager       *localRpc.RpcManager
	event            event.EventEmitter
	localRoomManager *LocalRoomManager
}

func (r *RemoteRpcRoomManager) getRpcByAddress(address *event.Address) (*localRpc.RpcClient, error) {
	rpc := r.rpcManager.GetRpcByAddress(address)
	if rpc == nil {
		return nil, fmt.Errorf("rpc client %s 不存在", address.MachineID)
	}

	return rpc, nil
}

func (r *RemoteRpcRoomManager) Start() {
	r.start()
	log.Info("remote rpc room manager start")
}

func (r *RemoteRpcRoomManager) GetRooms(ctx context.Context) ([]room.RemoteRoom, error) {
	rooms := make([]room.RemoteRoom, 0)
	for _, c := range r.rpcManager.GetRpcList() {
		req := NewRpcLocalRoomManagerRequest()
		res := NewRpcLocalRoomManagerResponse()
		err := c.Call(ctx, "LocalRpcRoomManager.GetRooms", req, res)
		if err != nil {
			return nil, err
		}

		rooms = append(rooms, res.GetRooms()...)
	}

	return rooms, nil
}

func (r *RemoteRpcRoomManager) GetRoomsByGroup(ctx context.Context, tags map[string]string) ([]room.RemoteRoom, error) {
	rooms := make([]room.RemoteRoom, 0)
	for _, c := range r.rpcManager.GetRpcList() {
		req := NewRpcLocalRoomManagerRequest()
		req.Tags = tags
		res := NewRpcLocalRoomManagerResponse()
		err := c.Call(ctx, "LocalRpcRoomManager.GetRoomsByGroup", req, res)
		if err != nil {
			return nil, err
		}

		rooms = append(rooms, res.GetRooms()...)
	}

	return rooms, nil
}

func (r *RemoteRpcRoomManager) ListRooms(ctx context.Context, tags map[string]string) ([]*room.Info, error) {
	rooms, err := r.GetRoomsByGroup(ctx, tags)
	if err != nil {
		return nil, err
	}

	infos := make([]*room.Info, 0)
	for _, r := range rooms {
		infos = append(infos, r.GetInfo())
	}

	sort.SliceStable(infos, func(i, j int) bool {
		return infos[i].CreatedAt.After(infos[j].CreatedAt)
	})

	return infos, nil
}

func (r *RemoteRpcRoomManager) CreateConnection() *room.Connection {
	address := r.AddressManager.GeneratorConnectionAddress()
	return &room.Connection{
		Address:   address,
		CreatedAt: time.Now(),
	}
}

func (r *RemoteRpcRoomManager) CreateRoom(ctx context.Context, info *room.Info) (room.Room, error) {
	return r.localRoomManager.CreateRoom(ctx, info)
}

func (r *RemoteRpcRoomManager) GetRoomUsers(ctx context.Context, info *room.Info) ([]*room.Connection, error) {
	room, err := r.GetRoom(ctx, info)
	if err != nil {
		return nil, err
	}

	if room == nil || reflect.ValueOf(room).IsNil() {
		return nil, fmt.Errorf("房间 %s 不存在呢", info.Address.ID)
	}

	return room.GetRoomUsers(), nil
}

func (r *RemoteRpcRoomManager) GetRoom(ctx context.Context, info *room.Info) (room.Room, error) {
	req := NewRpcLocalRoomManagerRequest()
	req.Info = info
	res := NewRpcLocalRoomManagerResponse()
	rpcClient, err := r.getRpcByAddress(info.Address)
	if err != nil {
		return nil, err
	}
	err = rpcClient.Call(ctx, "LocalRpcRoomManager.GetRoom", req, res)
	if err != nil {
		return nil, err
	}
	return res.Room, nil
}

func (r *RemoteRpcRoomManager) RemoveRoom(ctx context.Context, info *room.Info) error {
	req := NewRpcLocalRoomManagerRequest()
	req.Info = info
	res := NewRpcLocalRoomManagerResponse()
	rpcClient, err := r.getRpcByAddress(info.Address)
	if err != nil {
		return err
	}

	return rpcClient.Call(ctx, "LocalRpcRoomManager.RemoveRoom", req, res)
}

func (r *RemoteRpcRoomManager) getRemoteRoom(info *room.Info) (room.RemoteRoom, bool) {
	room, exist := r.getRoom(info)
	if !exist {
		return nil, false
	}
	return room.(*remoteRoom), true
}

func (r *RemoteRpcRoomManager) LeaveRoom(ctx context.Context, info *room.Info, connection *room.Connection) error {
	room, exist := r.getRemoteRoom(info)
	if exist {
		r.removeRoom(room)
		err := room.Close(ctx)
		if err != nil {
			log.Error("remote rpc room manager leaver room error %w", err)
		}
	}

	req := NewRpcLocalRoomManagerRequest()
	req.Info = info
	req.Connection = connection
	res := NewRpcLocalRoomManagerResponse()
	rpcClient, err := r.getRpcByAddress(info.Address)
	if err != nil {
		return err
	}

	return rpcClient.Call(ctx, "LocalRpcRoomManager.LeaveRoom", req, res)
}

func (r *RemoteRpcRoomManager) CreateAndJoinRoom(ctx context.Context, connection *room.Connection, opt *room.Info) (room.RemoteRoom, error) {
	room, err := r.CreateRoom(ctx, opt)
	if err != nil {
		return nil, err
	}

	remoteRoom, err := NewRemoteRoom(connection, opt, r.event, room)
	if err != nil {
		return nil, err
	}

	err = remoteRoom.Start(ctx)
	if err != nil {
		return nil, err
	}

	r.addRoom(remoteRoom)
	_, err = r.JoinRoom(ctx, connection, opt)
	if err != nil {
		return nil, err
	}

	return remoteRoom, nil
}

func (r *RemoteRpcRoomManager) JoinRoom(ctx context.Context, connection *room.Connection, opt *room.Info) (room.RemoteRoom, error) {
	room, err := r.GetRoom(ctx, opt)
	if err != nil {
		return nil, err
	}

	remoteRoom, err := NewRemoteRoom(connection, opt, r.event, room)
	if err != nil {
		return nil, err
	}

	err = remoteRoom.Start(ctx)
	if err != nil {
		return nil, err
	}

	err = r.joinRoom(ctx, connection, opt)
	if err != nil {
		return nil, err
	}

	return remoteRoom, nil
}

func (r *RemoteRpcRoomManager) joinRoom(ctx context.Context, connection *room.Connection, info *room.Info) error {
	req := NewRpcLocalRoomManagerRequest()
	req.Info = info
	req.Connection = connection
	res := NewRpcLocalRoomManagerResponse()
	rpcClient, err := r.getRpcByAddress(info.Address)
	if err != nil {
		return err
	}

	err = rpcClient.Call(ctx, "LocalRpcRoomManager.JoinRoom", req, res)
	if err != nil {
		return err
	}

	return nil
}
