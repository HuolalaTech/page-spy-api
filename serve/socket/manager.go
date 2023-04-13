package socket

import (
	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/event"
	"github.com/HuolalaTech/page-spy-api/logger"
	"github.com/HuolalaTech/page-spy-api/room"
	"github.com/HuolalaTech/page-spy-api/rpc"
)

func NewManager(config *config.Config) (*room.RemoteRpcRoomManager, error) {
	addressManager, err := rpc.NewAddressManager(config)
	if err != nil {
		return nil, err
	}

	rpcManager := rpc.NewRpcManager(addressManager)
	rpcManager.Run()
	localEvent := event.NewLocalEventEmitter(addressManager, rpcManager)
	localRoomManager := room.NewLocalRoomManager(localEvent, addressManager, 200)
	localRoomManager.Start()
	_, err = event.NewRpcEventEmitter(localEvent, rpcManager)
	if err != nil {
		return nil, err
	}

	_, err = room.NewLocalRpcRoomManager(localRoomManager, rpcManager)
	if err != nil {
		return nil, err
	}

	manager := room.NewRemoteRpcRoomManager(addressManager, rpcManager, localEvent, localRoomManager)
	manager.Start()
	logger.Log().Infof("init join serve %s ok", addressManager.GetSelfMachineID())
	return manager, nil
}
