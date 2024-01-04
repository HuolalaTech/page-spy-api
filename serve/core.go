package serve

import (
	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/container"
	"github.com/HuolalaTech/page-spy-api/util"
	"github.com/labstack/echo/v4"
)

func Run() {
	err := container.Container().Invoke(func(e *echo.Echo, config *config.Config, staticConfig *config.StaticConfig) {
		hash := staticConfig.GitHash
		version := staticConfig.Version
		if hash == "" {
			hash = "local"
		}
		if version == "" {
			version = "local"
		}
		log.Infof("server info: %s@%s", version, hash)
		log.Infof("LAN address http://%s:%s", util.GetLocalIP(), config.Port)
		log.Infof("Local address http://localhost:%s", config.Port)
		e.Logger.Fatal(e.Start(":" + config.Port))
	})

	if err != nil {
		log.Fatal(err)
	}
}
