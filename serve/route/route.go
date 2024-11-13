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

var blackTagName = []string{"page", "size", "from", "to"}

func include(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func getTags(params url.Values) []*storage.Tag {
	tags := []*storage.Tag{}
	for k, v := range params {
		if !include(blackTagName, k) {
			tags = append(tags, &storage.Tag{
				Key:   k,
				Value: strings.Join(v, " "),
			})
		}
	}

	return tags
}

func getQueryList(c echo.Context) (*data.FileListQuery, error) {
	page := c.QueryParam("page")
	size := c.QueryParam("size")
	if page == "" || size == "" {
		return nil, fmt.Errorf("find logs need page and size")
	}

	pageNum, err := strconv.Atoi(page)
	if err != nil {
		return nil, err
	}

	sizeNum, err := strconv.Atoi(size)
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("from time format error %w", err)
		}
		query.From = &fromStringUnix
	}

	toString := c.QueryParam("to")

	if toString != "" {
		toStringUnix, err := strconv.ParseInt(toString, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("to time format error %w", err)
		}

		query.To = &toStringUnix
	}
	return query, nil
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

	route.GET("/room/check", func(c echo.Context) error {
		socket.CheckRoomSecret(c.Response(), c.Request())
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

	route.GET("/logGroup/list", func(c echo.Context) error {
		query, err := getQueryList(c)
		if err != nil {
			return err
		}

		logGroups, err := core.GetLogGroupList(query)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(logGroups))
	})

	route.GET("/logGroup/files", func(c echo.Context) error {
		groupId := c.QueryParam("groupId")
		if groupId == "" {
			return fmt.Errorf("groupId is required")
		}

		logFiles, err := core.ListFilesInGroup(groupId)

		if err != nil {
			return err
		}
		return c.JSON(200, common.NewSuccessResponse(logFiles))
	})

	route.GET("/log/list", func(c echo.Context) error {
		query, err := getQueryList(c)
		if err != nil {
			return err
		}

		logs, err := core.GetFileList(query)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(logs))
	})

	route.DELETE("/log/delete", func(c echo.Context) error {
		if config.NotAllowedDeleteLog {
			return fmt.Errorf("not allowed delete log")
		}

		fileIds := c.QueryParams()["fileId"]
		for _, fileId := range fileIds {
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
		}

		return c.JSON(200, common.NewSuccessResponse(true))
	})

	route.DELETE("/logGroup/delete", func(c echo.Context) error {
		if config.NotAllowedDeleteLog {
			return fmt.Errorf("not allowed delete log")
		}

		groupIds := c.QueryParams()["groupId"]
		for _, groupId := range groupIds {
			err := core.DeleteLogGroup(groupId)
			if err != nil {
				return err
			}

		}

		return c.JSON(200, common.NewSuccessResponse(true))
	})

	route.POST("/logGroup/upload", func(c echo.Context) error {
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
		ts := getTags(c.QueryParams())

		groupId := c.QueryParam("groupId")
		if groupId == "" {
			return fmt.Errorf("groupId is required")
		}

		logFile := &storage.LogGroupFile{
			LogFile: storage.LogFile{
				Tags:       ts,
				Name:       file.Filename,
				Size:       file.Size,
				UpdateFile: fileBs,
			},
			GroupId: groupId,
		}

		createFile, err := core.CreateLogGroupFile(logFile)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(createFile))
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
			Tags:       getTags(c.QueryParams()),
			Name:       file.Filename,
			Size:       file.Size,
			UpdateFile: fileBs,
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
