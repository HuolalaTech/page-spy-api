package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	eventApi "github.com/HuolalaTech/page-spy-api/api/event"
	roomApi "github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/room"
	"github.com/labstack/echo/v4"
)

var supportedProtocolVersions = map[string]struct{}{
	"2025-11-25": {},
	"2025-06-18": {},
	"2025-03-26": {},
	"2024-11-05": {},
	"2024-10-07": {},
}

const defaultNegotiatedProtocolVersion = "2025-03-26"

type Handler struct {
	roomManager *room.RemoteRpcRoomManager
}

func NewHandler(roomManager *room.RemoteRpcRoomManager) *Handler {
	return &Handler{roomManager: roomManager}
}

func (h *Handler) Get(c echo.Context) error {
	return c.JSON(http.StatusMethodNotAllowed, newError(json.RawMessage("null"), -32000, "Method not allowed. Use POST for JSON responses.", nil))
}

func (h *Handler) Delete(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func (h *Handler) AnySubpath(c echo.Context) error {
	return c.JSON(http.StatusNotFound, newError(json.RawMessage("null"), -32000, "Invalid MCP endpoint. Use /mcp (or /api/v1/mcp) without extra path segments.", map[string]string{
		"path": c.Request().URL.Path,
	}))
}

func (h *Handler) Post(c echo.Context) error {
	accept := c.Request().Header.Get("Accept")
	if !strings.Contains(accept, "application/json") || !strings.Contains(accept, "text/event-stream") {
		return c.JSON(http.StatusNotAcceptable, newError(json.RawMessage("null"), -32000, "Not Acceptable: Client must accept both application/json and text/event-stream", nil))
	}
	ct := c.Request().Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		return c.JSON(http.StatusUnsupportedMediaType, newError(json.RawMessage("null"), -32000, "Unsupported Media Type: Content-Type must be application/json", nil))
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, newError(json.RawMessage("null"), -32700, "Parse error", err.Error()))
	}
	body = []byte(strings.TrimSpace(string(body)))
	if len(body) == 0 {
		return c.JSON(http.StatusBadRequest, newError(json.RawMessage("null"), -32600, "Invalid Request: Empty body", nil))
	}

	var raws []json.RawMessage
	if body[0] == '[' {
		if err := json.Unmarshal(body, &raws); err != nil {
			return c.JSON(http.StatusBadRequest, newError(json.RawMessage("null"), -32700, "Parse error", err.Error()))
		}
	} else {
		raws = []json.RawMessage{json.RawMessage(body)}
	}

	msgs := make([]jsonrpcMessage, 0, len(raws))
	for _, r := range raws {
		var m jsonrpcMessage
		if err := json.Unmarshal(r, &m); err != nil {
			return c.JSON(http.StatusBadRequest, newError(json.RawMessage("null"), -32700, "Parse error", err.Error()))
		}
		if m.JSONRPC == "" {
			m.JSONRPC = jsonrpcVersion
		}
		msgs = append(msgs, m)
	}

	hasRequests := false
	for _, m := range msgs {
		if len(m.ID) > 0 && m.Method != "" {
			hasRequests = true
			break
		}
	}
	if !hasRequests {
		return c.NoContent(http.StatusAccepted)
	}

	// Validate protocol version for non-initialize requests (lenient, but keep parity with SDK).
	protocolVersion := c.Request().Header.Get("mcp-protocol-version")
	if protocolVersion == "" {
		protocolVersion = defaultNegotiatedProtocolVersion
	}
	if _, ok := supportedProtocolVersions[protocolVersion]; !ok {
		return c.JSON(http.StatusBadRequest, newError(json.RawMessage("null"), -32000, "Bad Request: Unsupported protocol version", protocolVersion))
	}

	responses := make([]jsonrpcResponse, 0, len(msgs))
	for _, m := range msgs {
		if len(m.ID) == 0 || m.Method == "" {
			continue
		}
		responses = append(responses, h.handleRequest(c.Request().Context(), protocolVersion, m))
	}

	if len(responses) == 1 && body[0] != '[' {
		return c.JSON(http.StatusOK, responses[0])
	}
	return c.JSON(http.StatusOK, responses)
}

func (h *Handler) handleRequest(ctx context.Context, protocolVersion string, m jsonrpcMessage) jsonrpcResponse {
	switch m.Method {
	case "initialize":
		type initParams struct {
			ProtocolVersion string      `json:"protocolVersion"`
			Capabilities    interface{} `json:"capabilities"`
			ClientInfo      interface{} `json:"clientInfo"`
		}
		var p initParams
		_ = json.Unmarshal(m.Params, &p)
		v := p.ProtocolVersion
		if v == "" {
			v = defaultNegotiatedProtocolVersion
		}
		if _, ok := supportedProtocolVersions[v]; !ok {
			v = defaultNegotiatedProtocolVersion
		}
		return newResult(m.ID, map[string]interface{}{
			"protocolVersion": v,
			"capabilities": map[string]interface{}{
				"tools":     map[string]interface{}{},
				"resources": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "page-spy-api-mcp",
				"version": "0.0.0",
			},
			"instructions": "Use tools: list_rooms/read_room_debug_log/watch_room_debug_log/get_room_summary; resources under pagespy://rooms.",
		})
	case "ping":
		return newResult(m.ID, map[string]interface{}{})
	case "tools/list":
		return newResult(m.ID, map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{
					"name":        "list_rooms",
					"description": "List current rooms from page-spy-api",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"tags": map[string]interface{}{
								"type":                 "object",
								"additionalProperties": map[string]interface{}{"type": "string"},
							},
						},
						"additionalProperties": false,
					},
				},
				map[string]interface{}{
					"name":        "read_room_debug_log",
					"description": "Join room and collect role=client MESSAGE/BROADCAST events",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"address":   map[string]interface{}{"type": "string"},
							"secret":    map[string]interface{}{"type": "string"},
							"limit":     map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 2000},
							"timeoutMs": map[string]interface{}{"type": "integer", "minimum": 100, "maximum": 60000},
							"sinceId":   map[string]interface{}{"type": "string"},
							"format":    map[string]interface{}{"type": "string", "enum": []string{"text", "json"}},
							"type": map[string]interface{}{
								"type":  "array",
								"items": map[string]interface{}{"type": "string"},
							},
						},
						"required":             []string{"address"},
						"additionalProperties": false,
					},
				},
				map[string]interface{}{
					"name":        "watch_room_debug_log",
					"description": "Collect logs for a duration and return them",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"address":    map[string]interface{}{"type": "string"},
							"secret":     map[string]interface{}{"type": "string"},
							"limit":      map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 2000},
							"durationMs": map[string]interface{}{"type": "integer", "minimum": 100, "maximum": 60000},
							"format":     map[string]interface{}{"type": "string", "enum": []string{"text", "json"}},
							"type": map[string]interface{}{
								"type":  "array",
								"items": map[string]interface{}{"type": "string"},
							},
						},
						"required":             []string{"address"},
						"additionalProperties": false,
					},
				},
				map[string]interface{}{
					"name":        "get_room_summary",
					"description": "Compute a simple summary for recent room logs",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"address":   map[string]interface{}{"type": "string"},
							"secret":    map[string]interface{}{"type": "string"},
							"limit":     map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 2000},
							"timeoutMs": map[string]interface{}{"type": "integer", "minimum": 100, "maximum": 60000},
						},
						"required":             []string{"address"},
						"additionalProperties": false,
					},
				},
			},
		})
	case "tools/call":
		type callParams struct {
			Name      string      `json:"name"`
			Arguments interface{} `json:"arguments"`
		}
		var p callParams
		if err := json.Unmarshal(m.Params, &p); err != nil {
			return newError(m.ID, -32600, "Invalid Request: bad params", err.Error())
		}
		return h.callTool(ctx, p.Name, p.Arguments, m.ID)
	case "resources/list":
		return h.listResources(ctx, m.ID)
	case "resources/templates/list":
		return newResult(m.ID, map[string]interface{}{
			"resourceTemplates": []interface{}{
				map[string]interface{}{
					"uriTemplate": "pagespy://rooms/{address}",
					"description": "Get room info by address",
					"mimeType":    "application/json",
				},
				map[string]interface{}{
					"uriTemplate": "pagespy://rooms/{address}/debug-log{?limit,timeoutMs,sinceId,format,type,secret}",
					"description": "Read room debug log (role=client) via in-process room join + debugger-online",
					"mimeType":    "text/plain",
				},
			},
		})
	case "resources/read":
		type readParams struct {
			URI string `json:"uri"`
		}
		var p readParams
		if err := json.Unmarshal(m.Params, &p); err != nil {
			return newError(m.ID, -32600, "Invalid Request: bad params", err.Error())
		}
		return h.readResource(ctx, p.URI, m.ID)
	case "prompts/list":
		return newResult(m.ID, map[string]interface{}{"prompts": []interface{}{}})
	default:
		return newError(m.ID, -32601, "Method not found", m.Method)
	}
}

func (h *Handler) callTool(ctx context.Context, name string, args interface{}, id json.RawMessage) jsonrpcResponse {
	switch name {
	case "list_rooms":
		tags := parseTags(getMapField(args, "tags"))
		infos, err := h.roomManager.ListRooms(ctx, tags)
		if err != nil {
			return newError(id, -32000, "List rooms failed", err.Error())
		}
		rooms := make([]pageSpyRoomInfo, 0, len(infos))
		for _, i := range infos {
			rooms = append(rooms, toPageSpyRoomInfo(i))
		}
		bs, _ := json.MarshalIndent(rooms, "", "  ")
		return newResult(id, map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{"type": "text", "text": string(bs)},
			},
		})
	case "read_room_debug_log":
		address, _ := getStringField(args, "address")
		if _, err := parseAddress(address); err != nil {
			return newError(id, -32602, "Invalid params: bad address", err.Error())
		}
		secret, _ := getStringField(args, "secret")
		limit := getIntField(args, "limit", 200)
		timeoutMs := getIntField(args, "timeoutMs", 1200)
		sinceID, _ := getStringField(args, "sinceId")
		format, _ := getStringField(args, "format")
		if format == "" {
			format = "text"
		}
		types := getStringArrayField(args, "type")

		addr, _ := parseAddress(address)
		roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
		if err != nil {
			return newError(id, -32000, "Get room failed", err.Error())
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
			return newError(id, -32000, "Collect debug log failed", err.Error())
		}
		var text string
		if format == "json" {
			bs, _ := json.MarshalIndent(events, "", "  ")
			text = string(bs)
		} else {
			text = formatClientEventsText(events)
		}
		return newResult(id, map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{"type": "text", "text": text},
			},
		})
	case "watch_room_debug_log":
		address, _ := getStringField(args, "address")
		if _, err := parseAddress(address); err != nil {
			return newError(id, -32602, "Invalid params: bad address", err.Error())
		}
		secret, _ := getStringField(args, "secret")
		limit := getIntField(args, "limit", 200)
		durationMs := getIntField(args, "durationMs", 3000)
		format, _ := getStringField(args, "format")
		if format == "" {
			format = "text"
		}
		types := getStringArrayField(args, "type")

		addr, _ := parseAddress(address)
		roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
		if err != nil {
			return newError(id, -32000, "Get room failed", err.Error())
		}
		joinUserID := chooseJoinUserID(roomInfo.GetInfo())
		events, err := collectRoomDebugLog(ctx, h.roomManager, collectOptions{
			Address:    address,
			Secret:     secret,
			JoinUserID: joinUserID,
			SinceID:    "",
			Limit:      limit,
			Timeout:    time.Duration(durationMs) * time.Millisecond,
			Types:      types,
		})
		if err != nil {
			return newError(id, -32000, "Collect debug log failed", err.Error())
		}
		var text string
		if format == "json" {
			bs, _ := json.MarshalIndent(events, "", "  ")
			text = string(bs)
		} else {
			text = formatClientEventsText(events)
		}
		return newResult(id, map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{"type": "text", "text": text},
			},
		})
	case "get_room_summary":
		address, _ := getStringField(args, "address")
		if _, err := parseAddress(address); err != nil {
			return newError(id, -32602, "Invalid params: bad address", err.Error())
		}
		secret, _ := getStringField(args, "secret")
		limit := getIntField(args, "limit", 200)
		timeoutMs := getIntField(args, "timeoutMs", 1200)

		addr, _ := parseAddress(address)
		roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
		if err != nil {
			return newError(id, -32000, "Get room failed", err.Error())
		}
		joinUserID := chooseJoinUserID(roomInfo.GetInfo())
		events, err := collectRoomDebugLog(ctx, h.roomManager, collectOptions{
			Address:    address,
			Secret:     secret,
			JoinUserID: joinUserID,
			SinceID:    "",
			Limit:      limit,
			Timeout:    time.Duration(timeoutMs) * time.Millisecond,
		})
		if err != nil {
			return newError(id, -32000, "Collect debug log failed", err.Error())
		}
		bs, _ := json.MarshalIndent(roomSummary(events), "", "  ")
		return newResult(id, map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{"type": "text", "text": string(bs)},
			},
		})
	default:
		return newError(id, -32601, "Tool not found", name)
	}
}

func (h *Handler) listResources(ctx context.Context, id json.RawMessage) jsonrpcResponse {
	infos, err := h.roomManager.ListRooms(ctx, map[string]string{})
	if err != nil {
		return newError(id, -32000, "List rooms failed", err.Error())
	}
	resources := make([]interface{}, 0, 1+len(infos)*2)
	resources = append(resources, map[string]interface{}{
		"uri":         "pagespy://rooms",
		"description": "List current rooms",
		"mimeType":    "application/json",
	})
	for _, i := range infos {
		if i == nil || i.Address == nil {
			continue
		}
		addr := i.Address.ID
		resources = append(resources, map[string]interface{}{
			"uri":         "pagespy://rooms/" + addr,
			"description": "Room info",
			"mimeType":    "application/json",
		})
		resources = append(resources, map[string]interface{}{
			"uri":         "pagespy://rooms/" + addr + "/debug-log",
			"description": "Room debug log (role=client)",
			"mimeType":    "text/plain",
		})
	}
	return newResult(id, map[string]interface{}{"resources": resources})
}

func (h *Handler) readResource(ctx context.Context, uri string, id json.RawMessage) jsonrpcResponse {
	u, segs, err := parsePagespyResourceURI(uri)
	if err != nil {
		return newError(id, -32602, "Invalid params: bad uri", err.Error())
	}
	if u.Scheme != "pagespy" {
		return newError(id, -32602, "Invalid params: unsupported scheme", u.Scheme)
	}

	// pagespy://rooms
	if u.Host == "rooms" && strings.Trim(u.Path, "/") == "" {
		infos, err := h.roomManager.ListRooms(ctx, map[string]string{})
		if err != nil {
			return newError(id, -32000, "List rooms failed", err.Error())
		}
		rooms := make([]pageSpyRoomInfo, 0, len(infos))
		for _, i := range infos {
			rooms = append(rooms, toPageSpyRoomInfo(i))
		}
		bs, _ := json.MarshalIndent(rooms, "", "  ")
		return newResult(id, map[string]interface{}{
			"contents": []interface{}{
				map[string]interface{}{
					"uri":      uri,
					"mimeType": "application/json",
					"text":     string(bs),
				},
			},
		})
	}

	// pagespy:///rooms
	if u.Host == "" && len(segs) == 1 && segs[0] == "rooms" {
		infos, err := h.roomManager.ListRooms(ctx, map[string]string{})
		if err != nil {
			return newError(id, -32000, "List rooms failed", err.Error())
		}
		rooms := make([]pageSpyRoomInfo, 0, len(infos))
		for _, i := range infos {
			rooms = append(rooms, toPageSpyRoomInfo(i))
		}
		bs, _ := json.MarshalIndent(rooms, "", "  ")
		return newResult(id, map[string]interface{}{
			"contents": []interface{}{
				map[string]interface{}{
					"uri":      uri,
					"mimeType": "application/json",
					"text":     string(bs),
				},
			},
		})
	}

	// Accept both: pagespy://rooms/... and pagespy:///rooms/...
	parts := segs
	if len(parts) > 0 && parts[0] == "rooms" {
		parts = parts[1:]
	}
	if u.Host == "rooms" {
		// parsePagespyResourceURI includes host in segs, so parts should already exclude it.
	}

	if len(parts) == 0 {
		return newError(id, -32602, "Invalid params: missing resource path", uri)
	}

	// pagespy://rooms/{address}
	if u.Host == "rooms" && len(parts) == 1 {
		address := parts[0]
		addr, err := parseAddress(address)
		if err != nil {
			return newError(id, -32602, "Invalid params: bad address", err.Error())
		}
		roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
		if err != nil {
			return newError(id, -32000, "Get room failed", err.Error())
		}
		bs, _ := json.MarshalIndent(toPageSpyRoomInfo(roomInfo.GetInfo()), "", "  ")
		return newResult(id, map[string]interface{}{
			"contents": []interface{}{
				map[string]interface{}{
					"uri":      uri,
					"mimeType": "application/json",
					"text":     string(bs),
				},
			},
		})
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
			return newError(id, -32602, "Invalid params: bad address", err.Error())
		}
		roomInfo, err := h.roomManager.GetRoom(ctx, &roomApi.Info{Address: addr})
		if err != nil {
			return newError(id, -32000, "Get room failed", err.Error())
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
			return newError(id, -32000, "Collect debug log failed", err.Error())
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
		return newResult(id, map[string]interface{}{
			"contents": []interface{}{
				map[string]interface{}{
					"uri":      uri,
					"mimeType": mime,
					"text":     text,
				},
			},
		})
	}

	return newError(id, -32602, "Invalid params: unknown resource", uri)
}

func getMapField(v interface{}, key string) interface{} {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return m[key]
}

func getStringField(v interface{}, key string) (string, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return "", false
	}
	s, ok := m[key].(string)
	return s, ok
}

func getIntField(v interface{}, key string, def int) int {
	m, ok := v.(map[string]interface{})
	if !ok {
		return def
	}
	switch vv := m[key].(type) {
	case float64:
		return int(vv)
	case int:
		return vv
	default:
		return def
	}
}

func getStringArrayField(v interface{}, key string) []string {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	arr, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, it := range arr {
		s, ok := it.(string)
		if ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func parseAddress(id string) (*eventApi.Address, error) {
	return eventApi.NewAddressFromID(strings.TrimSpace(id))
}

func parseIntQuery(u *url.URL, key string, def int) int {
	v := u.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
