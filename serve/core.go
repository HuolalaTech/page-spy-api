package serve

import (
	"log"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/container"
	"github.com/labstack/echo/v4"
)

func Run() {
	err := container.Container().Invoke(func(e *echo.Echo, config *config.Config) {
		e.Logger.Fatal(e.Start(":" + config.Port))
	})

	if err != nil {
		log.Fatal(err)
	}
}
