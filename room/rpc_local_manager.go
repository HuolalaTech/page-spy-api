package room

import (
	"context"
	"net/http"
	"time"

	"github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/rpc"
)

type LocalRpcRoomManager struct {
	localRoomManager *LocalRoomManager
}

func NewLocalRpcRoomManager(localRoomManager *LocalRoomManager, rpcManager *rpc.RpcManager) (*LocalRpcRoomManager, error) {
	manager := &LocalRpcRoomManager{
		localRoomManager: localRoomManager,
	}

	return manager, rpcManager.Regist("LocalRpcRoomManager", manager)
}

type RpcLocalRoomManagerResponse struct {
	room.BasicRpcResponse
	Connection *room.Connection
	Rooms      []*localRoom
	Room       *localRoom
}

func NewRpcLocalRoomManagerResponse() *RpcLocalRoomManagerResponse {
	return &RpcLocalRoomManagerResponse{}
}

func (res *RpcLocalRoomManagerResponse) SetRooms(rooms []room.Room) {
	localRooms := make([]*localRoom, 0, len(rooms))
	for _, r := range rooms {
		localRooms = append(localRooms, r.(*localRoom))
	}
	res.Rooms = localRooms
}

func (res *RpcLocalRoomManagerResponse) GetRooms() []room.RemoteRoom {
	if len((res.Rooms)) <= 0 {
		return []room.RemoteRoom{}
	}

	rooms := make([]room.RemoteRoom, 0, len(res.Rooms))
	for _, r := range res.Rooms {
		rooms = append(rooms, r)
	}

	return rooms
}

type RpcLocalRoomManagerRequest struct {
	ContextTimeout int
	Group          string
	Info           *room.Info
	Connection     *room.Connection
}

func NewRpcLocalRoomManagerRequest() *RpcLocalRoomManagerRequest {
	return &RpcLocalRoomManagerRequest{
		ContextTimeout: 5,
	}
}

func (r *LocalRpcRoomManager) GetRoomsByGroup(_ *http.Request, req *RpcLocalRoomManagerRequest, res *RpcLocalRoomManagerResponse) error {
	rooms := r.localRoomManager.GetRoomsByGroup(req.Group)
	res.SetRooms(rooms)
	return nil
}

func (r *LocalRpcRoomManager) GetRooms(_ *http.Request, req *RpcLocalRoomManagerRequest, res *RpcLocalRoomManagerResponse) error {
	rooms := r.localRoomManager.GetRooms()
	res.SetRooms(rooms)
	return nil
}

func (r *LocalRpcRoomManager) CreateConnection(_ *http.Request, req *RpcLocalRoomManagerRequest, res *RpcLocalRoomManagerResponse) error {
	c, err := r.localRoomManager.CreateConnection()
	if err != nil {
		return res.SetError(err)
	}

	res.Connection = c
	return nil
}

func (r *LocalRpcRoomManager) CreateRoom(_ *http.Request, req *RpcLocalRoomManagerRequest, res *RpcLocalRoomManagerResponse) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.ContextTimeout)*time.Second)
	defer cancel()
	room, err := r.localRoomManager.CreateRoom(ctx, req.Info)
	if err != nil {
		return res.SetError(err)
	}

	res.Room = room.(*localRoom)
	return nil
}

func (r *LocalRpcRoomManager) GetRoom(_ *http.Request, req *RpcLocalRoomManagerRequest, res *RpcLocalRoomManagerResponse) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.ContextTimeout)*time.Second)
	defer cancel()
	room, err := r.localRoomManager.GetRoom(ctx, req.Info)
	if err != nil {
		return res.SetError(err)
	}

	res.Room = room.(*localRoom)
	return nil
}

func (r *LocalRpcRoomManager) RemoveRoom(_ *http.Request, req *RpcLocalRoomManagerRequest, res *RpcLocalRoomManagerResponse) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.ContextTimeout)*time.Second)
	defer cancel()
	return res.SetError(r.localRoomManager.RemoveRoom(ctx, req.Info))
}

func (r *LocalRpcRoomManager) LeaveRoom(_ *http.Request, req *RpcLocalRoomManagerRequest, res *RpcLocalRoomManagerResponse) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.ContextTimeout)*time.Second)
	defer cancel()
	return res.SetError(r.localRoomManager.LeaveRoom(ctx, req.Info, req.Connection))
}

func (r *LocalRpcRoomManager) JoinRoom(_ *http.Request, req *RpcLocalRoomManagerRequest, res *RpcLocalRoomManagerResponse) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.ContextTimeout)*time.Second)
	defer cancel()
	err := r.localRoomManager.JoinRoom(ctx, req.Info, req.Connection)
	return res.SetError(err)
}
