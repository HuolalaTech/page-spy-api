package container

import (
	"log"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/data"
	"github.com/HuolalaTech/page-spy-api/proxy"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/HuolalaTech/page-spy-api/serve/route"
	"github.com/HuolalaTech/page-spy-api/serve/socket"
	"github.com/HuolalaTech/page-spy-api/storage"
	"go.uber.org/dig"
)

func initContainer() (*dig.Container, error) {
	container := dig.New()
	err := container.Provide(config.LoadConfig)
	if err != nil {
		return nil, err
	}
	err = container.Provide(rpc.NewAddressManager)
	if err != nil {
		return nil, err
	}
	err = container.Provide(rpc.NewRpcManager)
	if err != nil {
		return nil, err
	}

	err = container.Provide(socket.NewManager)
	if err != nil {
		return nil, err
	}

	err = container.Provide(socket.NewWebSocket)
	if err != nil {
		return nil, err
	}

	err = container.Provide(data.NewData)

	if err != nil {
		return nil, err
	}
	err = container.Provide(storage.NewStorage)

	if err != nil {
		return nil, err
	}
	err = container.Provide(proxy.NewProxy)

	if err != nil {
		return nil, err
	}
	err = container.Provide(route.NewCore)

	if err != nil {
		return nil, err
	}
	err = container.Provide(route.NewEcho)

	return container, err
}

var container *dig.Container

func Container() *dig.Container {
	return container
}

func init() {
	var err error
	container, err = initContainer()
	if err != nil {
		log.Fatal(err)
	}
}
