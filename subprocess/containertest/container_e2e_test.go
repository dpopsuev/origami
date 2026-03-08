package containertest_test

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dpopsuev/origami/subprocess/containertest"
)

func repoRoot() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..")
}

// TestContainerE2E_BuildImages validates that the deploy/ Dockerfiles
// produce valid OCI images. Gated by podman availability.
func TestContainerE2E_BuildImages(t *testing.T) {
	env := containertest.NewEnv(t)
	root := repoRoot()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	images := []struct {
		dockerfile string
		tag        string
	}{
		{"deploy/Dockerfile.gateway", "origami-gateway-e2e"},
		{"deploy/Dockerfile.rca", "origami-rca-e2e"},
		{"deploy/Dockerfile.knowledge", "origami-knowledge-e2e"},
		{"deploy/Dockerfile.llm-worker", "origami-llm-worker-e2e"},
	}

	for _, img := range images {
		t.Run(img.tag, func(t *testing.T) {
			df := filepath.Join(root, img.dockerfile)
			env.BuildImageFromDockerfile(ctx, df, img.tag, root)
			t.Logf("built %s", img.tag)
		})
	}
}

// TestContainerE2E_GatewayKnowledge builds and starts the gateway +
// knowledge containers on host network, then validates tool routing.
// Uses host networking so containers can reach each other via localhost.
//
// Requires: podman, not -short.
func TestContainerE2E_GatewayKnowledge(t *testing.T) {
	env := containertest.NewEnv(t)
	root := repoRoot()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	t.Log("building knowledge image...")
	env.BuildImageFromDockerfile(ctx,
		filepath.Join(root, "deploy/Dockerfile.knowledge"),
		"origami-knowledge-e2e", root)

	t.Log("building gateway image...")
	env.BuildImageFromDockerfile(ctx,
		filepath.Join(root, "deploy/Dockerfile.gateway"),
		"origami-gateway-e2e", root)

	knPort := 19100
	gwPort := 19000

	t.Log("starting knowledge engine...")
	env.StartServiceWithConfig(ctx, containertest.ServiceConfig{
		Name:    "e2e-knowledge",
		Image:   "origami-knowledge-e2e",
		Port:    knPort,
		Network: "host",
		Args:    []string{"--port", fmt.Sprintf("%d", knPort)},
	})

	t.Log("starting gateway...")
	env.StartServiceWithConfig(ctx, containertest.ServiceConfig{
		Name:    "e2e-gateway",
		Image:   "origami-gateway-e2e",
		Port:    gwPort,
		Network: "host",
		Args: []string{
			"--port", fmt.Sprintf("%d", gwPort),
			"--backend", fmt.Sprintf("knowledge=http://127.0.0.1:%d/mcp", knPort),
		},
	})

	t.Run("HealthProbes", func(t *testing.T) {
		for _, path := range []string{"/healthz", "/readyz"} {
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", gwPort, path))
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s = %d, want 200", path, resp.StatusCode)
			}
		}
	})

	t.Run("ToolRouting", func(t *testing.T) {
		transport := &sdkmcp.StreamableClientTransport{
			Endpoint: fmt.Sprintf("http://127.0.0.1:%d/mcp", gwPort),
		}
		client := sdkmcp.NewClient(
			&sdkmcp.Implementation{Name: "e2e-client", Version: "v0.1.0"},
			nil,
		)
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		defer session.Close()

		tools, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}

		hasKnowledge := false
		for _, tool := range tools.Tools {
			if tool.Name == "knowledge_search" || tool.Name == "knowledge_read" {
				hasKnowledge = true
			}
		}
		if !hasKnowledge {
			t.Error("missing knowledge tools through gateway")
		}
		t.Logf("discovered %d tools through gateway", len(tools.Tools))
	})
}
