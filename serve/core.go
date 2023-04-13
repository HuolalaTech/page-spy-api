package serve

import (
	"io/fs"
	"log"
	"net/http"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/container"
	selfMiddleware "github.com/HuolalaTech/page-spy-api/serve/middleware"
	"github.com/HuolalaTech/page-spy-api/serve/socket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type StaticConfig struct {
	DirName string
	Files   fs.FS
}

func Run(staticConfig *StaticConfig) {
	container := container.Container()
	err := container.Invoke(func(socket *socket.WebSocket, config *config.Config) {
		e := echo.New()
		e.Use(selfMiddleware.Logger())
		e.Use(selfMiddleware.CORS(config))
		route := e.Group("/api/v1")
		route.GET("/room/list", func(c echo.Context) error {
			socket.ListRooms(c.Response(), c.Request())
			return nil
		})

		route.POST("/room/create", func(c echo.Context) error {
			socket.CreateRoom(c.Response(), c.Request())
			return nil
		})

		route.GET("/room/create", func(c echo.Context) error {
			socket.CreateRoom(c.Response(), c.Request())
			return nil
		})

		route.GET("/ws/room/join", func(c echo.Context) error {
			socket.JoinRoom(c.Response(), c.Request())
			return nil
		})

		if staticConfig != nil {
			e.GET("/*", echo.WrapHandler(http.FileServer(http.FS(&FallbackFS{FS: staticConfig.Files, Fallback: staticConfig.DirName + "/index.html"}))), middleware.Rewrite(map[string]string{"/*": "/dist/$1"}))
		}

		e.Logger.Fatal(e.Start(":" + config.Port))
	})

	if err != nil {
		log.Fatal(err)
	}
}
