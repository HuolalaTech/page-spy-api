package serve

import (
	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/container"
	"github.com/HuolalaTech/page-spy-api/util"
	"github.com/labstack/echo/v4"
)

func Run() {
	err := container.Container().Invoke(func(e *echo.Echo, config *config.Config) {
		log.Infof("远程访问地址 http://%s:%s", util.GetLocalIP(), config.Port)
		log.Infof("本地访问地址 http://localhost:%s", config.Port)
		e.Logger.Fatal(e.Start(":" + config.Port))
	})

	if err != nil {
		log.Fatal(err)
	}
}
