package container

import (
	"log"
	"net/http"

	"github.com/HuolalaTech/page-spy-api/config"
	selfMiddleware "github.com/HuolalaTech/page-spy-api/serve/middleware"
	"github.com/HuolalaTech/page-spy-api/serve/socket"
	"github.com/HuolalaTech/page-spy-api/static"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

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

	err = container.Provide(func(socket *socket.WebSocket, config *config.Config, staticConfig *config.StaticConfig) *echo.Echo {
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

		route.GET("/ws/room/join", func(c echo.Context) error {
			socket.JoinRoom(c.Response(), c.Request())
			return nil
		})

		if staticConfig != nil {
			e.GET("/*", echo.WrapHandler(http.FileServer(http.FS(&static.FallbackFS{FS: staticConfig.Files, Fallback: staticConfig.DirName + "/index.html"}))), middleware.Rewrite(map[string]string{"/*": "/dist/$1"}))
		}

		return e
	})

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
