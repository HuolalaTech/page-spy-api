package rpc

import (
	"fmt"
	"net/http"

	"github.com/HuolalaTech/page-spy-api/api/event"
	"github.com/gorilla/mux"
	hRpc "github.com/gorilla/rpc/v2"
	hJson "github.com/gorilla/rpc/v2/json"
)

type RpcManager struct {
	addressManager *AddressManager
	rpcList        map[string]*RpcClient
	server         *hRpc.Server
}

func NewRpcManager(addressManager *AddressManager) *RpcManager {
	rpcList := make(map[string]*RpcClient)
	for machineID, address := range addressManager.GetMachineIpInfo() {
		rpcList[machineID] = NewRpcClient(address.Ip + ":" + address.Port)
	}
	server := hRpc.NewServer()
	server.RegisterCodec(hJson.NewCodec(), "application/json")
	return &RpcManager{
		addressManager: addressManager,
		rpcList:        rpcList,
		server:         server,
	}
}

func (r *RpcManager) GetRpcByAddress(address *event.Address) *RpcClient {
	return r.rpcList[address.MachineID]
}

func (r *RpcManager) GetRpcList() []*RpcClient {
	list := make([]*RpcClient, 0, len(r.rpcList))
	for _, l := range r.rpcList {
		list = append(list, l)
	}

	return list
}

func (r *RpcManager) Regist(name string, api interface{}) error {
	return r.server.RegisterService(api, name)
}

func (r *RpcManager) listen() error {
	route := mux.NewRouter()
	route.Handle("/rpc", r.server)
	err := http.ListenAndServe(":"+r.addressManager.GetSelfAddress().Port, route)
	if err != nil {
		return fmt.Errorf("监听端口错误 %w", err)
	}
	return nil
}

func (r *RpcManager) Run() {
	go func() {
		err := r.listen()
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("启动本地rpc服务端口:%s", r.addressManager.GetSelfAddress().Port)
	}()
}
