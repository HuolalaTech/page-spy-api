package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	eventApi "github.com/HuolalaTech/page-spy-api/api/event"
	roomApi "github.com/HuolalaTech/page-spy-api/api/room"
	"github.com/HuolalaTech/page-spy-api/room"
)

const (
	defaultDebuggerUserID = "Debugger"
	defaultClientUserID   = "Client"
	defaultMcpUserID      = "MCP"
)

type pageSpyRoomInfo struct {
	Address     string              `json:"address"`
	Group       string              `json:"group"`
	Name        string              `json:"name"`
	UseSecret   bool                `json:"useSecret"`
	Tags        map[string]string   `json:"tags"`
	ActiveAt    time.Time           `json:"activeAt"`
	CreatedAt   time.Time           `json:"createdAt"`
	Connections []pageSpyConnection `json:"connections"`
}

type pageSpyConnection struct {
	Address   string    `json:"address"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func toPageSpyRoomInfo(i *roomApi.Info) pageSpyRoomInfo {
	out := pageSpyRoomInfo{
		Address:     "",
		Group:       i.Group,
		Name:        i.Name,
		UseSecret:   i.UseSecret,
		Tags:        i.Tags,
		ActiveAt:    i.ActiveAt,
		CreatedAt:   i.CreatedAt,
		Connections: make([]pageSpyConnection, 0),
	}
	if i.Address != nil {
		out.Address = i.Address.ID
	}
	for _, c := range i.Connections {
		if c == nil || c.Address == nil {
			continue
		}
		out.Connections = append(out.Connections, pageSpyConnection{
			Address:   c.Address.ID,
			UserID:    c.UserID,
			Name:      c.Name,
			CreatedAt: c.CreatedAt,
		})
	}
	return out
}

func hasUser(i *roomApi.Info, userID string) bool {
	for _, c := range i.Connections {
		if c != nil && c.UserID == userID {
			return true
		}
	}
	return false
}

func chooseJoinUserID(i *roomApi.Info) string {
	if hasUser(i, defaultDebuggerUserID) {
		return defaultMcpUserID
	}
	return defaultDebuggerUserID
}

type clientDebugEvent struct {
	WsType      string          `json:"wsType"`      // message|broadcast
	MessageType string          `json:"messageType"` // e.g. console/network/page...
	ID          string          `json:"id"`
	CreatedAt   int64           `json:"createdAt"` // ms
	Item        json.RawMessage `json:"item"`      // raw SpyMessage.MessageItem
}

type spyMessageItem struct {
	Type string `json:"type"`
	Role string `json:"role"`
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

type collectOptions struct {
	Address    string
	Secret     string
	JoinUserID string
	SinceID    string
	Limit      int
	Timeout    time.Duration
	Types      []string
}

func collectRoomDebugLog(ctx context.Context, roomManager *room.RemoteRpcRoomManager, opt collectOptions) ([]clientDebugEvent, error) {
	if opt.Limit <= 0 {
		opt.Limit = 200
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 1200 * time.Millisecond
	}

	address, err := eventApi.NewAddressFromID(strings.TrimSpace(opt.Address))
	if err != nil {
		return nil, err
	}

	roomOpt := &roomApi.Info{
		Address: address,
		Secret:  opt.Secret,
	}

	connection := roomManager.CreateConnection()
	connection.UserID = opt.JoinUserID
	connection.Name = opt.JoinUserID

	r, err := roomManager.JoinRoom(ctx, connection, roomOpt)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = r.Close(context.Background(), "mcp_close")
		_ = roomManager.LeaveRoom(context.Background(), roomOpt, connection)
	}()

	users, err := roomManager.GetRoomUsers(ctx, roomOpt)
	if err != nil {
		return nil, err
	}
	var clientConn *roomApi.Connection
	for _, u := range users {
		if u != nil && u.UserID == defaultClientUserID {
			clientConn = u
			break
		}
	}
	if clientConn != nil {
		latestID := opt.SinceID
		payload := map[string]interface{}{
			"type": "debugger-online",
			"data": map[string]interface{}{
				"latestId": latestID,
			},
		}
		bs, _ := json.Marshal(payload)
		msg := roomApi.NewSendMessage(json.RawMessage(bs), connection, clientConn)
		_ = r.SendMessage(ctx, msg)
	}

	typeSet := map[string]struct{}{}
	for _, t := range opt.Types {
		tt := strings.TrimSpace(t)
		if tt == "" {
			continue
		}
		typeSet[tt] = struct{}{}
	}

	seen := map[string]struct{}{}
	events := make([]clientDebugEvent, 0, opt.Limit)

	timeoutCtx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()

	for {
		if len(events) >= opt.Limit {
			return events, nil
		}
		select {
		case <-timeoutCtx.Done():
			return events, nil
		case msg := <-r.OnMessage():
			if msg == nil {
				continue
			}
			switch msg.Type {
			case roomApi.MessageType:
				content, ok := msg.Content.(*roomApi.MessageMessageContent)
				if !ok {
					continue
				}
				ev, ok := convertClientEvent("message", msg.CreatedAt, content.Data, typeSet, opt.SinceID)
				if !ok {
					continue
				}
				if _, dup := seen[ev.ID]; dup {
					continue
				}
				seen[ev.ID] = struct{}{}
				events = append(events, ev)
			case roomApi.BroadcastType:
				content, ok := msg.Content.(*roomApi.BroadcastMessageContent)
				if !ok {
					continue
				}
				ev, ok := convertClientEvent("broadcast", msg.CreatedAt, content.Data, typeSet, opt.SinceID)
				if !ok {
					continue
				}
				if _, dup := seen[ev.ID]; dup {
					continue
				}
				seen[ev.ID] = struct{}{}
				events = append(events, ev)
			}
		}
	}
}

func convertClientEvent(wsType string, createdAt int64, raw json.RawMessage, typeSet map[string]struct{}, sinceID string) (clientDebugEvent, bool) {
	var item spyMessageItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return clientDebugEvent{}, false
	}
	if item.Role != "client" {
		return clientDebugEvent{}, false
	}
	if len(typeSet) > 0 {
		if _, ok := typeSet[item.Type]; !ok {
			return clientDebugEvent{}, false
		}
	}
	if item.Data.ID == "" {
		return clientDebugEvent{}, false
	}
	if sinceID != "" && item.Data.ID == sinceID {
		return clientDebugEvent{}, false
	}
	if createdAt <= 0 {
		createdAt = time.Now().UnixNano() / int64(time.Millisecond)
	}
	return clientDebugEvent{
		WsType:      wsType,
		MessageType: item.Type,
		ID:          item.Data.ID,
		CreatedAt:   createdAt,
		Item:        raw,
	}, true
}

func formatClientEventsText(events []clientDebugEvent) string {
	var b strings.Builder
	for i, e := range events {
		ts := time.UnixMilli(e.CreatedAt).Format(time.RFC3339Nano)
		line := fmt.Sprintf("%s %s %s id=%s\n", ts, e.WsType, e.MessageType, e.ID)
		if i == len(events)-1 {
			b.WriteString(strings.TrimRight(line, "\n"))
		} else {
			b.WriteString(line)
		}
	}
	return b.String()
}

func parseTags(v interface{}) map[string]string {
	out := map[string]string{}
	m, ok := v.(map[string]interface{})
	if !ok {
		return out
	}
	for k, vv := range m {
		s, ok := vv.(string)
		if !ok {
			continue
		}
		out[k] = s
	}
	return out
}

func parsePagespyResourceURI(uri string) (*url.URL, []string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, nil, err
	}
	segs := []string{}
	if u.Host != "" {
		segs = append(segs, u.Host)
	}
	for _, s := range strings.Split(u.Path, "/") {
		if s == "" {
			continue
		}
		segs = append(segs, s)
	}
	return u, segs, nil
}

func roomSummary(events []clientDebugEvent) map[string]interface{} {
	countByType := map[string]int{}
	for _, e := range events {
		countByType[e.MessageType] = countByType[e.MessageType] + 1
	}
	keys := make([]string, 0, len(countByType))
	for k := range countByType {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := map[string]int{}
	for _, k := range keys {
		ordered[k] = countByType[k]
	}

	var latestAt interface{} = nil
	var latestID interface{} = nil
	if len(events) > 0 {
		latestAt = events[len(events)-1].CreatedAt
		latestID = events[len(events)-1].ID
	}
	return map[string]interface{}{
		"total":       len(events),
		"countByType": ordered,
		"latestId":    latestID,
		"latestAt":    latestAt,
	}
}
