package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type HTTPClient struct {
	serverURL      string
	httpClient     *http.Client
	sessionID      string
	requestContext *RequestContext
	
	requestID atomic.Int64
	
	pendingMu sync.Mutex
	pending   map[int64]chan *JSONRPCResponse
	
	ctx    context.Context
	cancel context.CancelFunc
}

func NewHTTPClient(serverURL string, requestContext *RequestContext) *HTTPClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &HTTPClient{
		serverURL:      serverURL,
		httpClient:     &http.Client{Timeout: 60 * time.Second},
		requestContext: requestContext,
		pending:        make(map[int64]chan *JSONRPCResponse),
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (c *HTTPClient) Start(ctx context.Context, binary string, args []string) error {
	return nil
}

func (c *HTTPClient) Initialize(ctx context.Context) (*InitializeResult, error) {
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "initialize",
		Params: InitializeRequest{
			ProtocolVersion: "2024-11-05",
			Capabilities: Capabilities{
				Tools: &ToolCapabilities{
					ListChanged: false,
				},
			},
			ClientInfo: ClientInfo{
				Name:    "aks-mcp-bench",
				Version: "0.1.0",
			},
		},
	}
	
	resp, err := c.call(ctx, req)
	if err != nil {
		return nil, err
	}
	
	if resp.Error != nil {
		return nil, fmt.Errorf("initialize error: %s", resp.Error.Message)
	}
	
	var result InitializeResult
	if err := remarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse initialize result: %w", err)
	}
	
	notification := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	
	if err := c.send(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}
	
	return &result, nil
}

func (c *HTTPClient) ListTools(ctx context.Context) ([]Tool, error) {
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/list",
	}
	
	resp, err := c.call(ctx, req)
	if err != nil {
		return nil, err
	}
	
	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}
	
	var result ListToolsResult
	if err := remarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools list result: %w", err)
	}
	
	return result.Tools, nil
}

func (c *HTTPClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*CallToolResult, error) {
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/call",
		Params: CallToolRequest{
			Name:      name,
			Arguments: arguments,
		},
	}
	
	resp, err := c.call(ctx, req)
	if err != nil {
		return nil, err
	}
	
	if resp.Error != nil {
		return nil, fmt.Errorf("tools/call error [%d]: %s", resp.Error.Code, resp.Error.Message)
	}
	
	var result CallToolResult
	if err := remarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool call result: %w", err)
	}
	
	return &result, nil
}

func (c *HTTPClient) Close() error {
	c.cancel()
	
	if c.sessionID != "" {
		req, err := http.NewRequest(http.MethodDelete, c.serverURL+"/mcp", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Mcp-Session-Id", c.sessionID)
		
		c.httpClient.Do(req)
	}
	
	return nil
}

func (c *HTTPClient) call(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL+"/mcp", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	if c.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	
	if c.requestContext != nil {
		contextData, err := json.Marshal(c.requestContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request context: %w", err)
		}
		httpReq.Header.Set("X-Request-Context", string(contextData))
	}
	
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer httpResp.Body.Close()
	
	if c.sessionID == "" {
		c.sessionID = httpResp.Header.Get("Mcp-Session-Id")
	}
	
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(body))
	}
	
	var resp JSONRPCResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &resp, nil
}

func (c *HTTPClient) send(ctx context.Context, req *JSONRPCRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL+"/mcp", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	if c.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	
	if c.requestContext != nil {
		contextData, err := json.Marshal(c.requestContext)
		if err != nil {
			return fmt.Errorf("failed to marshal request context: %w", err)
		}
		httpReq.Header.Set("X-Request-Context", string(contextData))
	}
	
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer httpResp.Body.Close()
	
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(body))
	}
	
	return nil
}

func (c *HTTPClient) nextID() int64 {
	return c.requestID.Add(1)
}
