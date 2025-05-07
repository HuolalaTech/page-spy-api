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

	// 公共路由 - 无需认证
	publicRoute := route.Group("")

	// 认证相关API
	publicRoute.POST("/auth/verify", func(c echo.Context) error {
		type PasswordRequest struct {
			Password string `json:"password"`
		}

		// 解析请求体中的密码
		var passwordReq PasswordRequest
		if err := c.Bind(&passwordReq); err != nil {
			return c.JSON(http.StatusBadRequest, common.NewErrorResponseWithCode("Invalid request format", "INVALID_REQUEST"))
		}

		// 检查是否设置了密码
		if !selfMiddleware.IsPasswordSet(config) {
			return c.JSON(http.StatusOK, common.NewErrorResponseWithCode("System password not set, please set a password first", "PASSWORD_REQUIRED"))
		}

		// 验证密码
		if !selfMiddleware.VerifyPassword(config, passwordReq.Password) {
			return c.JSON(http.StatusOK, common.NewErrorResponseWithCode("Incorrect password", "INVALID_PASSWORD"))
		}

		// 生成JWT令牌
		token, expirationHours, err := selfMiddleware.GenerateToken(config)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, common.NewErrorResponseWithCode("Failed to generate token", "TOKEN_GENERATION_FAILED"))
		}

		return c.JSON(http.StatusOK, common.NewSuccessResponse(map[string]interface{}{
			"success":   true,
			"message":   "Authentication successful",
			"token":     token,
			"expiresIn": expirationHours * 3600, // 过期时间，单位秒
		}))
	})

	// 设置密码接口
	publicRoute.POST("/auth/set-password", func(c echo.Context) error {
		type PasswordRequest struct {
			Password string `json:"password"`
		}

		// 解析请求体中的密码
		var passwordReq PasswordRequest
		if err := c.Bind(&passwordReq); err != nil {
			return c.JSON(http.StatusBadRequest, common.NewErrorResponseWithCode("Invalid request format", "INVALID_REQUEST"))
		}

		// 检查是否已经设置了密码
		if selfMiddleware.IsPasswordSet(config) && !selfMiddleware.IsFirstStart(config) {
			return c.JSON(http.StatusOK, common.NewErrorResponseWithCode("Password already set, cannot set again", "PASSWORD_ALREADY_SET"))
		}

		// 密码验证逻辑
		if passwordReq.Password == "" {
			return c.JSON(http.StatusOK, common.NewErrorResponseWithCode("Password cannot be empty", "INVALID_PASSWORD"))
		}

		// 设置密码
		err := selfMiddleware.SetPassword(config, passwordReq.Password)
		if err != nil {
			// 如果是因为环境变量设置了密码导致的错误
			if httpErr, ok := err.(*echo.HTTPError); ok {
				return c.JSON(http.StatusOK, common.NewErrorResponseWithCode(httpErr.Message.(string), "ENV_PASSWORD_SET"))
			}
			return c.JSON(http.StatusInternalServerError, common.NewErrorResponseWithCode("Failed to set password", "PASSWORD_SET_FAILED"))
		}

		// 生成JWT令牌
		token, expirationHours, err := selfMiddleware.GenerateToken(config)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, common.NewErrorResponseWithCode("Failed to generate token", "TOKEN_GENERATION_FAILED"))
		}

		return c.JSON(http.StatusOK, common.NewSuccessResponse(map[string]interface{}{
			"success":            true,
			"message":            "Password set successfully",
			"token":              token,
			"passwordConfigured": true,
			"expiresIn":          expirationHours * 3600, // 过期时间，单位秒
		}))
	})

	// 跳过密码设置接口
	publicRoute.POST("/auth/skip-password", func(c echo.Context) error {
		// 检查是否已经设置了密码
		if selfMiddleware.IsPasswordSet(config) && !selfMiddleware.IsFirstStart(config) {
			return c.JSON(http.StatusOK, common.NewErrorResponseWithCode("Password already set, cannot skip password setup", "PASSWORD_ALREADY_SET"))
		}

		// 跳过密码设置
		err := selfMiddleware.SkipPasswordSetup(config)
		if err != nil {
			// 如果是因为环境变量设置了密码导致的错误
			if httpErr, ok := err.(*echo.HTTPError); ok {
				return c.JSON(http.StatusOK, common.NewErrorResponseWithCode(httpErr.Message.(string), "ENV_PASSWORD_SET"))
			}
			return c.JSON(http.StatusInternalServerError, common.NewErrorResponseWithCode("Failed to skip password setup", "PASSWORD_SKIP_FAILED"))
		}

		// 无密码模式下，我们不生成token（因为无需认证）
		return c.JSON(http.StatusOK, common.NewSuccessResponse(map[string]interface{}{
			"success":            true,
			"message":            "Password setup skipped successfully",
			"passwordConfigured": false,
			"noPasswordMode":     true,
		}))
	})

	// 认证状态接口
	publicRoute.GET("/auth/status", func(c echo.Context) error {
		return c.JSON(http.StatusOK, common.NewSuccessResponse(map[string]interface{}{
			"passwordConfigured": selfMiddleware.IsPasswordSet(config),
			"isFirstStart":       selfMiddleware.IsFirstStart(config),
		}))
	})

	// 无需认证的公共接口
	publicRoute.POST("/room/create", func(c echo.Context) error {
		socket.CreateRoom(c.Response(), c.Request())
		return nil
	})

	publicRoute.GET("/ws/room/join", func(c echo.Context) error {
		socket.JoinRoom(c.Response(), c.Request())
		return nil
	})

	publicRoute.GET("/room/check", func(c echo.Context) error {
		socket.CheckRoomSecret(c.Response(), c.Request())
		return nil
	})

	// 受保护的路由组 - 需要认证
	protectedRoute := route.Group("")
	protectedRoute.Use(selfMiddleware.Auth(config))

	// 需要认证的API
	protectedRoute.GET("/room/list", func(c echo.Context) error {
		socket.ListRooms(c.Response(), c.Request())
		return nil
	})

	protectedRoute.GET("/log/count", func(c echo.Context) error {
		key := c.QueryParam("key")
		result, err := core.data.CountLogsGroup(key)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(result))
	})

	protectedRoute.GET("/log/download", func(c echo.Context) error {
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

	protectedRoute.GET("/logGroup/list", func(c echo.Context) error {
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

	protectedRoute.GET("/logGroup/files", func(c echo.Context) error {
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

	protectedRoute.GET("/log/list", func(c echo.Context) error {
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

	protectedRoute.DELETE("/log/delete", func(c echo.Context) error {
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

	protectedRoute.DELETE("/logGroup/delete", func(c echo.Context) error {
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

	// 以下是需要公开的上传接口
	publicRoute.POST("/logGroup/upload", func(c echo.Context) error {
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

	publicRoute.POST("/jsonLog/upload", func(c echo.Context) error {
		fileName := c.QueryParam("name")
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return fmt.Errorf("open upload file error: %w", err)
		}

		logFile := &storage.LogFile{
			Tags:       getTags(c.QueryParams()),
			Name:       fileName,
			Size:       int64(len(body)),
			UpdateFile: body,
		}

		createFile, err := core.CreateFile(logFile)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(createFile))
	})

	publicRoute.POST("/log/upload", func(c echo.Context) error {
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
