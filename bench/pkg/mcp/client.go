package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Client represents an MCP client that communicates with an MCP server via stdio
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	
	// Request ID counter
	requestID atomic.Int64
	
	// Pending requests
	pendingMu sync.Mutex
	pending   map[int64]chan *JSONRPCResponse
	
	// Read loop control
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewClient creates a new MCP client
func NewClient() *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		pending: make(map[int64]chan *JSONRPCResponse),
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
}

// Start starts the MCP server process
func (c *Client) Start(ctx context.Context, binary string, args []string) error {
	c.cmd = exec.CommandContext(ctx, binary, args...)
	
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	
	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	
	// Start reading responses
	go c.readLoop()
	
	return nil
}

// Initialize performs the MCP initialize handshake
func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
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
	
	// Send initialized notification
	notification := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	
	if err := c.send(notification); err != nil {
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}
	
	return &result, nil
}

// ListTools returns the list of available tools
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
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

// CallTool executes a tool via MCP
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*CallToolResult, error) {
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

// Close closes the MCP client and stops the server
func (c *Client) Close() error {
	c.cancel()
	
	if c.stdin != nil {
		c.stdin.Close()
	}
	
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	
	<-c.done
	return nil
}

// call sends a request and waits for response
func (c *Client) call(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	id, ok := req.ID.(int64)
	if !ok {
		return nil, fmt.Errorf("invalid request ID type")
	}
	
	// Register pending request
	respCh := make(chan *JSONRPCResponse, 1)
	c.pendingMu.Lock()
	c.pending[id] = respCh
	c.pendingMu.Unlock()
	
	// Clean up on exit
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()
	
	// Send request
	if err := c.send(req); err != nil {
		return nil, err
	}
	
	// Wait for response
	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.ctx.Done():
		return nil, fmt.Errorf("client closed")
	}
}

// send sends a JSON-RPC request/notification
func (c *Client) send(req *JSONRPCRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	data = append(data, '\n')
	
	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}
	
	return nil
}

// readLoop reads responses from stdout
func (c *Client) readLoop() {
	defer close(c.done)
	
	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		
		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			// Log error but continue
			continue
		}
		
		// Deliver response to pending request
		id, ok := resp.ID.(float64)
		if !ok {
			continue
		}
		
		reqID := int64(id)
		c.pendingMu.Lock()
		ch, exists := c.pending[reqID]
		c.pendingMu.Unlock()
		
		if exists {
			select {
			case ch <- &resp:
			case <-c.ctx.Done():
				return
			}
		}
	}
}

// nextID generates the next request ID
func (c *Client) nextID() int64 {
	return c.requestID.Add(1)
}

// remarshal converts interface{} to specific type via JSON
func remarshal(from interface{}, to interface{}) error {
	data, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, to)
}
