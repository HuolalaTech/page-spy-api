package container

import (
	"io/fs"
	"log"
	"net/http"

	"github.com/HuolalaTech/page-spy-api/config"
	selfMiddleware "github.com/HuolalaTech/page-spy-api/serve/middleware"
	"github.com/HuolalaTech/page-spy-api/serve/socket"
	"github.com/HuolalaTech/page-spy-api/static"
	"github.com/labstack/echo/v4"
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
		e.HidePort = true
		e.HideBanner = true
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
			dist, err := fs.Sub(staticConfig.Files, "dist")
			if err != nil {
				// it will never be here
				panic(err)
			}

			ff := static.NewFallbackFS(
				dist,
				"index.html",
			)

			e.GET(
				"/*",
				echo.WrapHandler(
					http.FileServer(http.FS(ff))),
				selfMiddleware.Cache(),
			)
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
