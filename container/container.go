package container

import (
	"log"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/serve/socket"
	"go.uber.org/dig"
)

func initContainer() (*dig.Container, error) {
	container := dig.New()
	err := container.Provide(config.LoadConfig)
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

	return container, nil
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
