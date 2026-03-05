package subprocess

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ContainerManager manages OCI containers via podman or docker.
type ContainerManager struct {
	Runtime string // "podman" or "docker"; defaults to "podman"

	mu         sync.RWMutex
	containers map[string]*Container
}

// Container represents a running OCI container.
type Container struct {
	ID    string
	Image string
	Port  int

	session *sdkmcp.ClientSession
}

// NewContainerManager creates a ContainerManager with the given runtime.
func NewContainerManager(runtime string) *ContainerManager {
	if runtime == "" {
		runtime = "podman"
	}
	return &ContainerManager{
		Runtime:    runtime,
		containers: make(map[string]*Container),
	}
}

// Start pulls/builds the image if needed, starts a container, and connects
// an MCP client via TCP.
func (cm *ContainerManager) Start(ctx context.Context, name, image string, port int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.containers[name]; exists {
		return fmt.Errorf("container %q already running", name)
	}

	// Start the container
	args := []string{
		"run", "-d",
		"--name", name,
		"-p", fmt.Sprintf("%d:%d", port, 9100),
		image,
	}

	cmd := exec.CommandContext(ctx, cm.Runtime, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start container %q: %s: %w", name, strings.TrimSpace(string(output)), err)
	}

	containerID := strings.TrimSpace(string(output))

	c := &Container{
		ID:    containerID,
		Image: image,
		Port:  port,
	}

	// Wait for the container to be ready and connect MCP client
	if err := cm.connectMCP(ctx, c); err != nil {
		// Clean up the container if connection fails
		cm.stopContainer(ctx, containerID)
		return fmt.Errorf("connect MCP to container %q: %w", name, err)
	}

	cm.containers[name] = c
	return nil
}

// Stop stops and removes a container.
func (cm *ContainerManager) Stop(ctx context.Context, name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.containers[name]
	if !ok {
		return nil
	}

	if c.session != nil {
		c.session.Close()
	}

	if err := cm.stopContainer(ctx, c.ID); err != nil {
		return err
	}

	delete(cm.containers, name)
	return nil
}

// Swap replaces a running container with a new image.
func (cm *ContainerManager) Swap(ctx context.Context, name, newImage string) error {
	if err := cm.Stop(ctx, name); err != nil {
		return fmt.Errorf("stop for swap: %w", err)
	}

	cm.mu.RLock()
	port := 9100 // default
	cm.mu.RUnlock()

	return cm.Start(ctx, name, newImage, port)
}

// CallTool calls a tool on a container's MCP server.
func (cm *ContainerManager) CallTool(ctx context.Context, name string, tool string, args map[string]any) (*sdkmcp.CallToolResult, error) {
	cm.mu.RLock()
	c, ok := cm.containers[name]
	cm.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown container %q", name)
	}
	if c.session == nil {
		return nil, fmt.Errorf("container %q not connected", name)
	}
	return c.session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      tool,
		Arguments: args,
	})
}

func (cm *ContainerManager) stopContainer(ctx context.Context, id string) error {
	// Stop
	stop := exec.CommandContext(ctx, cm.Runtime, "stop", "-t", "5", id)
	stop.CombinedOutput()

	// Remove
	rm := exec.CommandContext(ctx, cm.Runtime, "rm", "-f", id)
	output, err := rm.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove container %s: %s: %w", id, strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (cm *ContainerManager) connectMCP(ctx context.Context, c *Container) error {
	addr := "localhost:" + strconv.Itoa(c.Port)

	// Poll until the HTTP endpoint is ready (max 10s).
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	endpoint := "http://" + addr + "/mcp"

	transport := &sdkmcp.StreamableClientTransport{
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "origami-container-client", Version: "v0.1.0"},
		nil,
	)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("MCP connect to %s: %w", endpoint, err)
	}

	c.session = session
	return nil
}
