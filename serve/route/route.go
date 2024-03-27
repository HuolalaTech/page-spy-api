package route

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/data"
	"github.com/HuolalaTech/page-spy-api/proxy"
	"github.com/HuolalaTech/page-spy-api/serve/common"
	selfMiddleware "github.com/HuolalaTech/page-spy-api/serve/middleware"
	"github.com/HuolalaTech/page-spy-api/serve/socket"
	"github.com/HuolalaTech/page-spy-api/static"
	"github.com/HuolalaTech/page-spy-api/storage"
	"github.com/labstack/echo/v4"
)

var TagName = []string{"project", "title", "deviceId", "userAgent"}

func include(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func getTags(params url.Values) []*data.Tag {
	tags := []*data.Tag{}
	for k, v := range params {
		if include(TagName, k) {
			tags = append(tags, &data.Tag{
				Key:   k,
				Value: strings.Join(v, " "),
			})
		}
	}

	return tags
}

func NewEcho(socket *socket.WebSocket, core *CoreApi, config *config.Config, proxyManager *proxy.ProxyManager, staticConfig *config.StaticConfig) *echo.Echo {
	e := echo.New()
	e.Use(selfMiddleware.Logger())
	e.Use(selfMiddleware.Error())
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

	route.GET("/log/download", func(c echo.Context) error {
		fileId := c.QueryParam("fileId")
		machine, err := core.GetMachineIdByFileName(fileId)
		if err != nil {
			return err
		}
		if !core.IsSelfMachine(machine) {
			return proxyManager.Proxy(machine, c)
		}

		file, err := core.GetFile(fileId)
		if err != nil {
			return err
		}

		defer file.FileSteam.Close()
		c.Response().Header().Set("Content-Disposition", "attachment; filename="+file.Name)
		c.Response().Header().Set("Content-Type", "application/octet-stream")
		c.Response().Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))

		_, err = io.Copy(c.Response().Writer, file.FileSteam)
		if err != nil {
			return err
		}

		return nil
	})

	route.GET("/log/list", func(c echo.Context) error {
		page := c.QueryParam("page")
		size := c.QueryParam("size")
		if page == "" || size == "" {
			return fmt.Errorf("find logs need page and size")
		}

		pageNum, err := strconv.Atoi(page)
		if err != nil {
			return err
		}

		sizeNum, err := strconv.Atoi(size)
		if err != nil {
			return err
		}

		query := &data.FileListQuery{
			PageQuery: data.PageQuery{
				Size: sizeNum,
				Page: pageNum,
			},
			Tags: getTags(c.QueryParams()),
		}

		fromString := c.QueryParam("from")
		if fromString != "" {
			fromStringUnix, err := strconv.ParseInt(fromString, 10, 64)
			if err != nil {
				return fmt.Errorf("from time format error %w", err)
			}
			query.From = &fromStringUnix
		}

		toString := c.QueryParam("to")

		if toString != "" {
			toStringUnix, err := strconv.ParseInt(toString, 10, 64)
			if err != nil {
				return fmt.Errorf("to time format error %w", err)
			}

			query.To = &toStringUnix
		}

		logs, err := core.GetFileList(query)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(logs))
	})

	route.DELETE("/log/delete", func(c echo.Context) error {
		fileId := c.QueryParam("fileId")
		machine, err := core.GetMachineIdByFileName(fileId)
		if err != nil {
			return err
		}
		if !core.IsSelfMachine(machine) {
			return proxyManager.Proxy(machine, c)
		}

		err = core.DeleteFile(fileId)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(true))
	})

	route.POST("/log/upload", func(c echo.Context) error {
		file, err := c.FormFile("log")
		if err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("open upload file error: %w", err)
		}

		defer src.Close()
		fileBs, err := io.ReadAll(src)
		if err != nil {
			return fmt.Errorf("read upload file error: %w", err)
		}

		logFile := &storage.LogFile{
			Tags: getTags(c.QueryParams()),
			Name: file.Filename,
			Size: file.Size,
			File: fileBs,
		}

		createFile, err := core.CreateFile(logFile)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(createFile))
	})

	if staticConfig != nil {
		dist, err := fs.Sub(staticConfig.Files, "dist")
		if err != nil {
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
}
