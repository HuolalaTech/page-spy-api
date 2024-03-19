package socket

import (
	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/event"
	"github.com/HuolalaTech/page-spy-api/logger"
	"github.com/HuolalaTech/page-spy-api/room"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/HuolalaTech/page-spy-api/util"
)

func NewManager(config *config.Config, addressManager *rpc.AddressManager) (*room.RemoteRpcRoomManager, error) {
	rpcManager := rpc.NewRpcManager(addressManager)
	rpcManager.Run()
	localEvent := event.NewLocalEventEmitter(addressManager, rpcManager)
	localRoomManager := room.NewLocalRoomManager(localEvent, addressManager, 200)
	localRoomManager.Start()
	_, err := event.NewRpcEventEmitter(localEvent, rpcManager)
	if err != nil {
		return nil, err
	}

	_, err = room.NewLocalRpcRoomManager(localRoomManager, rpcManager)
	if err != nil {
		return nil, err
	}

	manager := room.NewRemoteRpcRoomManager(addressManager, rpcManager, localEvent, localRoomManager)
	manager.Start()
	logger.Log().Infof("start rpc server %s successful", addressManager.GetSelfMachineID())
	logger.Log().Infof("local ip %s:%s", util.GetLocalIP(), config.Port)
	return manager, nil
}
