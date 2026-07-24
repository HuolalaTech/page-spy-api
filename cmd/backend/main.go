package main

import (
	"log"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/container"
	"github.com/HuolalaTech/page-spy-api/serve"
)

func main() {
	appContainer := container.Container()
	if err := appContainer.Provide(func() *config.StaticConfig {
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	serve.Run()
}
