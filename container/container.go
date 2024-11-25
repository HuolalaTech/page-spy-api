package container

import (
	"log"
	"sync"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/data"
	"github.com/HuolalaTech/page-spy-api/proxy"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/HuolalaTech/page-spy-api/serve/route"
	"github.com/HuolalaTech/page-spy-api/serve/socket"
	"github.com/HuolalaTech/page-spy-api/storage"
	"github.com/HuolalaTech/page-spy-api/task"
	"go.uber.org/dig"
)

func InitContainer() (*dig.Container, error) {
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

	err = container.Provide(task.NewTaskManager)
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

var once sync.Once

func Container() *dig.Container {
	if container == nil {
		once.Do(func() {
			InitDefault()
		})
	}

	return container
}

func SetContainer(c *dig.Container) {
	container = c
}

func InitDefault() {
	var err error
	container, err = InitContainer()
	if err != nil {
		log.Fatal(err)
	}
}
