package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	roomApi "github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/room"
	"github.com/labstack/echo/v4"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Handler struct {
	roomManager *room.RemoteRpcRoomManager
	httpHandler http.Handler
}

func NewHandler(roomManager *room.RemoteRpcRoomManager) *Handler {
	h := &Handler{roomManager: roomManager}

	hooks := &server.Hooks{}
	mcpServer := server.NewMCPServer(
		"page-spy-api-mcp",
		"0.0.0",
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(false, false),
		server.WithInstructions("Use tools: list_rooms/read_room_debug_log/watch_room_debug_log/get_room_summary; resources under pagespy://rooms."),
		server.WithRecovery(),
		server.WithResourceRecovery(),
		server.WithHooks(hooks),
	)

	mcpServer.AddTool(
		mcp.NewTool(
			"list_rooms",
			mcp.WithDescription("List current rooms from page-spy-api"),
			mcp.WithInputSchema[listRoomsArgs](),
		),
		h.handleListRooms,
	)
	mcpServer.AddTool(
		mcp.NewTool(
			"read_room_debug_log",
			mcp.WithDescription("Join room and collect role=client MESSAGE/BROADCAST events"),
			mcp.WithInputSchema[readRoomDebugLogArgs](),
		),
		h.handleReadRoomDebugLog,
	)
	mcpServer.AddTool(
		mcp.NewTool(
			"watch_room_debug_log",
			mcp.WithDescription("Collect logs for a duration and return them"),
			mcp.WithInputSchema[watchRoomDebugLogArgs](),
		),
		h.handleWatchRoomDebugLog,
	)
	mcpServer.AddTool(
		mcp.NewTool(
			"get_room_summary",
			mcp.WithDescription("Compute a simple summary for recent room logs"),
			mcp.WithInputSchema[getRoomSummaryArgs](),
		),
		h.handleGetRoomSummary,
	)

	mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"pagespy://rooms/{address}",
			"pagespy_room_info",
			mcp.WithTemplateDescription("Get room info by address"),
			mcp.WithTemplateMIMEType("application/json"),
		),
		h.readResourceByURI,
	)
	mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"pagespy://rooms/{address}/debug-log{?limit,timeoutMs,sinceId,format,type,secret}",
			"pagespy_room_debug_log",
			mcp.WithTemplateDescription("Read room debug log (role=client) via in-process room join + debugger-online"),
			mcp.WithTemplateMIMEType("text/plain"),
		),
		h.readResourceByURI,
	)

	hooks.AddBeforeListResources(func(ctx context.Context, _ any, _ *mcp.ListResourcesRequest) {
		h.setDynamicResources(ctx, mcpServer)
	})
	h.setDynamicResources(context.Background(), mcpServer)

	h.httpHandler = server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true))
	return h
}

func (h *Handler) Get(c echo.Context) error {
	h.httpHandler.ServeHTTP(c.Response().Writer, c.Request())
	return nil
}

func (h *Handler) Delete(c echo.Context) error {
	h.httpHandler.ServeHTTP(c.Response().Writer, c.Request())
	return nil
}

func (h *Handler) AnySubpath(c echo.Context) error {
	return c.JSON(http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    -32000,
			"message": "Invalid MCP endpoint. Use /mcp (or /api/v1/mcp) without extra path segments.",
			"data": map[string]any{
				"path": c.Request().URL.Path,
			},
		},
	})
}

func (h *Handler) Post(c echo.Context) error {
	h.httpHandler.ServeHTTP(c.Response().Writer, c.Request())
	return nil
}

type listRoomsArgs struct {
	Tags map[string]string `json:"tags,omitempty"`
}

type readRoomDebugLogArgs struct {
	Address   string   `json:"address"`
	Secret    string   `json:"secret,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	TimeoutMs int      `json:"timeoutMs,omitempty"`
	SinceID   string   `json:"sinceId,omitempty"`
	Format    string   `json:"format,omitempty"`
	Type      []string `json:"type,omitempty"`
}

type watchRoomDebugLogArgs struct {
	Address    string   `json:"address"`
	Secret     string   `json:"secret,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	DurationMs int      `json:"durationMs,omitempty"`
	Format     string   `json:"format,omitempty"`
	Type       []string `json:"type,omitempty"`
}

type getRoomSummaryArgs struct {
	Address   string `json:"address"`
	Secret    string `json:"secret,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	TimeoutMs int    `json:"timeoutMs,omitempty"`
}

func toolErrorf(format string, args ...any) *mcp.CallToolResult {
	r := mcp.NewToolResultText(fmt.Sprintf(format, args...))
	r.IsError = true
	return r
}

func (h *Handler) handleListRooms(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var a listRoomsArgs
	_ = request.BindArguments(&a)
	tags := a.Tags
	if tags == nil {
		tags = map[string]string{}
	}

	infos, err := h.roomManager.ListRooms(ctx, tags)
	if err != nil {
		return toolErrorf("List rooms failed: %v", err), nil
	}
	rooms := make([]pageSpyRoomInfo, 0, len(infos))
	for _, i := range infos {
		rooms = append(rooms, toPageSpyRoomInfo(i))
	}
	bs, err := json.MarshalIndent(rooms, "", "  ")
	if err != nil {
		return toolErrorf("Marshal result failed: %v", err), nil
	}
	return mcp.NewToolResultStructured(map[string]any{"rooms": rooms}, string(bs)), nil
}

func (h *Handler) handleReadRoomDebugLog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var a readRoomDebugLogArgs
	if err := request.BindArguments(&a); err != nil {
		return toolErrorf("Invalid args: %v", err), nil
	}
	if _, err := parseAddress(a.Address); err != nil {
		return toolErrorf("Invalid params: bad address: %v", err), nil
	}
	if a.Limit <= 0 {
		a.Limit = 200
	}
	if a.TimeoutMs <= 0 {
		a.TimeoutMs = 1200
	}
	if a.Format == "" {
		a.Format = "text"
	}

	addr, _ := parseAddress(a.Address)
	roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
	if err != nil {
		return toolErrorf("Get room failed: %v", err), nil
	}
	joinUserID := chooseJoinUserID(roomInfo.GetInfo())
	events, err := collectRoomDebugLog(ctx, h.roomManager, collectOptions{
		Address:    a.Address,
		Secret:     a.Secret,
		JoinUserID: joinUserID,
		SinceID:    a.SinceID,
		Limit:      a.Limit,
		Timeout:    time.Duration(a.TimeoutMs) * time.Millisecond,
		Types:      a.Type,
	})
	if err != nil {
		return toolErrorf("Collect debug log failed: %v", err), nil
	}

	if a.Format == "json" {
		bs, err := json.MarshalIndent(events, "", "  ")
		if err != nil {
			return toolErrorf("Marshal result failed: %v", err), nil
		}
		return mcp.NewToolResultStructured(map[string]any{"events": events}, string(bs)), nil
	}
	return mcp.NewToolResultText(formatClientEventsText(events)), nil
}

func (h *Handler) handleWatchRoomDebugLog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var a watchRoomDebugLogArgs
	if err := request.BindArguments(&a); err != nil {
		return toolErrorf("Invalid args: %v", err), nil
	}
	if _, err := parseAddress(a.Address); err != nil {
		return toolErrorf("Invalid params: bad address: %v", err), nil
	}
	if a.Limit <= 0 {
		a.Limit = 200
	}
	if a.DurationMs <= 0 {
		a.DurationMs = 3000
	}
	if a.Format == "" {
		a.Format = "text"
	}

	addr, _ := parseAddress(a.Address)
	roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
	if err != nil {
		return toolErrorf("Get room failed: %v", err), nil
	}
	joinUserID := chooseJoinUserID(roomInfo.GetInfo())
	events, err := collectRoomDebugLog(ctx, h.roomManager, collectOptions{
		Address:    a.Address,
		Secret:     a.Secret,
		JoinUserID: joinUserID,
		SinceID:    "",
		Limit:      a.Limit,
		Timeout:    time.Duration(a.DurationMs) * time.Millisecond,
		Types:      a.Type,
	})
	if err != nil {
		return toolErrorf("Collect debug log failed: %v", err), nil
	}

	if a.Format == "json" {
		bs, err := json.MarshalIndent(events, "", "  ")
		if err != nil {
			return toolErrorf("Marshal result failed: %v", err), nil
		}
		return mcp.NewToolResultStructured(map[string]any{"events": events}, string(bs)), nil
	}
	return mcp.NewToolResultText(formatClientEventsText(events)), nil
}

func (h *Handler) handleGetRoomSummary(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var a getRoomSummaryArgs
	if err := request.BindArguments(&a); err != nil {
		return toolErrorf("Invalid args: %v", err), nil
	}
	if _, err := parseAddress(a.Address); err != nil {
		return toolErrorf("Invalid params: bad address: %v", err), nil
	}
	if a.Limit <= 0 {
		a.Limit = 200
	}
	if a.TimeoutMs <= 0 {
		a.TimeoutMs = 1200
	}

	addr, _ := parseAddress(a.Address)
	roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
	if err != nil {
		return toolErrorf("Get room failed: %v", err), nil
	}
	joinUserID := chooseJoinUserID(roomInfo.GetInfo())
	events, err := collectRoomDebugLog(ctx, h.roomManager, collectOptions{
		Address:    a.Address,
		Secret:     a.Secret,
		JoinUserID: joinUserID,
		SinceID:    "",
		Limit:      a.Limit,
		Timeout:    time.Duration(a.TimeoutMs) * time.Millisecond,
	})
	if err != nil {
		return toolErrorf("Collect debug log failed: %v", err), nil
	}
	res, err := mcp.NewToolResultJSON(roomSummary(events))
	if err != nil {
		return toolErrorf("Marshal result failed: %v", err), nil
	}
	return res, nil
}

func (h *Handler) setDynamicResources(ctx context.Context, mcpServer *server.MCPServer) {
	base := server.ServerResource{
		Resource: mcp.NewResource(
			"pagespy://rooms",
			"pagespy_rooms",
			mcp.WithResourceDescription("List current rooms"),
			mcp.WithMIMEType("application/json"),
		),
		Handler: h.readResourceByURI,
	}

	resources := []server.ServerResource{base}

	infos, err := h.roomManager.ListRooms(ctx, map[string]string{})
	if err == nil {
		for _, i := range infos {
			if i == nil || i.Address == nil {
				continue
			}
			addr := i.Address.ID
			resources = append(resources,
				server.ServerResource{
					Resource: mcp.NewResource(
						"pagespy://rooms/"+addr,
						"pagespy_room_"+addr,
						mcp.WithResourceDescription("Room info"),
						mcp.WithMIMEType("application/json"),
					),
					Handler: h.readResourceByURI,
				},
				server.ServerResource{
					Resource: mcp.NewResource(
						"pagespy://rooms/"+addr+"/debug-log",
						"pagespy_room_debug_log_"+addr,
						mcp.WithResourceDescription("Room debug log (role=client)"),
						mcp.WithMIMEType("text/plain"),
					),
					Handler: h.readResourceByURI,
				},
			)
		}
	}

	mcpServer.SetResources(resources...)
}

func (h *Handler) readResourceByURI(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	uri := request.Params.URI
	u, segs, err := parsePagespyResourceURI(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "pagespy" {
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	// pagespy://rooms
	if u.Host == "rooms" && strings.Trim(u.Path, "/") == "" {
		infos, err := h.roomManager.ListRooms(ctx, map[string]string{})
		if err != nil {
			return nil, err
		}
		rooms := make([]pageSpyRoomInfo, 0, len(infos))
		for _, i := range infos {
			rooms = append(rooms, toPageSpyRoomInfo(i))
		}
		bs, _ := json.MarshalIndent(rooms, "", "  ")
		return []mcp.ResourceContents{
			mcp.TextResourceContents{URI: uri, MIMEType: "application/json", Text: string(bs)},
		}, nil
	}

	// pagespy:///rooms
	if u.Host == "" && len(segs) == 1 && segs[0] == "rooms" {
		infos, err := h.roomManager.ListRooms(ctx, map[string]string{})
		if err != nil {
			return nil, err
		}
		rooms := make([]pageSpyRoomInfo, 0, len(infos))
		for _, i := range infos {
			rooms = append(rooms, toPageSpyRoomInfo(i))
		}
		bs, _ := json.MarshalIndent(rooms, "", "  ")
		return []mcp.ResourceContents{
			mcp.TextResourceContents{URI: uri, MIMEType: "application/json", Text: string(bs)},
		}, nil
	}

	// Accept both: pagespy://rooms/... and pagespy:///rooms/...
	parts := segs
	if len(parts) > 0 && parts[0] == "rooms" {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("missing resource path: %s", uri)
	}

	// pagespy://rooms/{address}
	if u.Host == "rooms" && len(parts) == 1 {
		address := parts[0]
		addr, err := parseAddress(address)
		if err != nil {
			return nil, err
		}
		roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
		if err != nil {
			return nil, err
		}
		bs, _ := json.MarshalIndent(toPageSpyRoomInfo(roomInfo.GetInfo()), "", "  ")
		return []mcp.ResourceContents{
			mcp.TextResourceContents{URI: uri, MIMEType: "application/json", Text: string(bs)},
		}, nil
	}

	// pagespy://rooms/{address}/debug-log?...params
	if u.Host == "rooms" && len(parts) >= 2 && parts[1] == "debug-log" {
		address := parts[0]
		limit := parseIntQuery(u, "limit", 200)
		timeoutMs := parseIntQuery(u, "timeoutMs", 1200)
		sinceID := u.Query().Get("sinceId")
		format := u.Query().Get("format")
		if format == "" {
			format = "text"
		}
		secret := u.Query().Get("secret")
		types := u.Query()["type"]

		addr, err := parseAddress(address)
		if err != nil {
			return nil, err
		}
		roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
		if err != nil {
			return nil, err
		}
		joinUserID := chooseJoinUserID(roomInfo.GetInfo())
		events, err := collectRoomDebugLog(ctx, h.roomManager, collectOptions{
			Address:    address,
			Secret:     secret,
			JoinUserID: joinUserID,
			SinceID:    sinceID,
			Limit:      limit,
			Timeout:    time.Duration(timeoutMs) * time.Millisecond,
			Types:      types,
		})
		if err != nil {
			return nil, err
		}

		mime := "text/plain"
		var text string
		if format == "json" {
			mime = "application/json"
			bs, _ := json.MarshalIndent(events, "", "  ")
			text = string(bs)
		} else {
			text = formatClientEventsText(events)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{URI: uri, MIMEType: mime, Text: text},
		}, nil
	}

	return nil, fmt.Errorf("unknown resource: %s", uri)
}
