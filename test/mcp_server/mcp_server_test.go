package mcp_server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func liveEnabled(t *testing.T) bool {
	t.Helper()
	if os.Getenv("MCP_TEST_LIVE") == "1" {
		return true
	}
	t.Skip("跳过：未设置环境变量 MCP_TEST_LIVE=1（这些测试需要本地运行中的 page-spy-api）")
	return false
}

func baseURL() string {
	v := strings.TrimSpace(os.Getenv("MCP_BASE_URL"))
	if v == "" {
		return "http://127.0.0.1:6752"
	}
	return strings.TrimRight(v, "/")
}

func endpointURL() string {
	v := strings.TrimSpace(os.Getenv("MCP_ENDPOINT"))
	if v == "" {
		return baseURL() + "/mcp"
	}
	return strings.TrimRight(v, "/")
}

func authHeader() string {
	return strings.TrimSpace(os.Getenv("MCP_AUTH"))
}

func postInitialize(ctx context.Context, endpoint string) (int, string, error) {
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "pagespy-mcp-probe-go",
				"version": "0.0.0",
			},
		},
	}
	bs, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bs))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Protocol-Version", "2025-03-26")
	if ah := authHeader(); ah != "" {
		req.Header.Set("Authorization", ah)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	text, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(text), nil
}

func TestProbeEndpointCandidates(t *testing.T) {
	if !liveEnabled(t) {
		return
	}

	base := baseURL()
	candidates := []string{
		base + "/mcp",
		base + "/mcp/",
		base + "/api/v1/mcp",
		base + "/api/v1/mcp/",
	}

	var lastErr error
	for _, ep := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		code, body, err := postInitialize(ctx, ep)
		cancel()

		if err != nil {
			lastErr = err
			t.Logf("probe %s error: %v", ep, err)
			continue
		}
		t.Logf("probe %s status=%d body(0..300)=%q", ep, code, strings.TrimSpace(body)[:min(300, len(strings.TrimSpace(body)))])
		if code >= 200 && code < 300 {
			return
		}
		lastErr = fmt.Errorf("status=%d body=%s", code, body)
	}

	if lastErr != nil {
		t.Fatalf("no working MCP endpoint found under %s: %v", base, lastErr)
	}
	t.Fatalf("no working MCP endpoint found under %s", base)
}

func TestStreamableHTTPClientConnectAndListTools(t *testing.T) {
	if !liveEnabled(t) {
		return
	}

	headers := map[string]string{}
	if ah := authHeader(); ah != "" {
		headers["Authorization"] = ah
	}

	tr, err := transport.NewStreamableHTTP(endpointURL(), transport.WithHTTPHeaders(headers))
	if err != nil {
		t.Fatalf("NewStreamableHTTP: %v", err)
	}
	c := mcpclient.NewClient(tr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("client.Start: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2025-03-26",
			ClientInfo:      mcp.Implementation{Name: "page-spy-api-mcp-test-go", Version: "0.0.0"},
			Capabilities:    mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		t.Fatalf("client.Initialize: %v", err)
	}

	tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatalf("expected tools, got empty list")
	}
}

func TestSmokeToolsAndResources(t *testing.T) {
	if !liveEnabled(t) {
		return
	}

	headers := map[string]string{}
	if ah := authHeader(); ah != "" {
		headers["Authorization"] = ah
	}

	tr, err := transport.NewStreamableHTTP(endpointURL(), transport.WithHTTPHeaders(headers))
	if err != nil {
		t.Fatalf("NewStreamableHTTP: %v", err)
	}
	c := mcpclient.NewClient(tr)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("client.Start: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2025-03-26",
			ClientInfo:      mcp.Implementation{Name: "page-spy-api-mcp-smoke-go", Version: "0.0.0"},
			Capabilities:    mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		t.Fatalf("client.Initialize: %v", err)
	}

	if _, err := c.ListResources(ctx, mcp.ListResourcesRequest{}); err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if _, err := c.ListResourceTemplates(ctx, mcp.ListResourceTemplatesRequest{}); err != nil {
		t.Fatalf("ListResourceTemplates: %v", err)
	}

	roomsRes, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_rooms",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(list_rooms): %v", err)
	}
	if roomsRes.IsError {
		t.Fatalf("CallTool(list_rooms) returned isError")
	}

	roomText := firstText(roomsRes)
	var rooms []struct {
		Address string `json:"address"`
	}
	_ = json.Unmarshal([]byte(roomText), &rooms)

	address := strings.TrimSpace(os.Getenv("MCP_ROOM_ADDRESS"))
	if address == "" && len(rooms) > 0 {
		address = rooms[0].Address
	}
	secret := strings.TrimSpace(os.Getenv("MCP_ROOM_SECRET"))

	if address == "" {
		t.Skip("跳过：未找到房间地址（可设置 MCP_ROOM_ADDRESS/MCP_ROOM_SECRET 以测试 read_room_debug_log）")
		return
	}

	_, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "read_room_debug_log",
			Arguments: map[string]any{
				"address":   address,
				"secret":    secret,
				"timeoutMs": 800,
				"limit":     20,
				"format":    "json",
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(read_room_debug_log): %v", err)
	}
}

func firstText(res *mcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	for _, c := range res.Content {
		switch v := c.(type) {
		case mcp.TextContent:
			return v.Text
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

