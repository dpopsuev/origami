package harvester_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	skn "github.com/dpopsuev/origami/schematics/harvester"
	"github.com/dpopsuev/origami/schematics/toolkit"
)

func newHarvesterServeMux(router *skn.AccessRouter) *http.ServeMux {
	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test-harvester", Version: "v0.1.0"},
		nil,
	)
	skn.RegisterTools(server, router)

	mcpHandler := sdkmcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *sdkmcp.Server { return server },
		&sdkmcp.StreamableHTTPOptions{Stateless: true},
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if router.Ready() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	return mux
}

func TestHealthz_ReturnsOK(t *testing.T) {
	router := skn.NewRouter()
	mux := newHarvesterServeMux(router)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestReadyz_NoDrivers_Returns503(t *testing.T) {
	router := skn.NewRouter()
	mux := newHarvesterServeMux(router)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("readyz status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestReadyz_WithDriver_ReturnsOK(t *testing.T) {
	driver := &stubDriver{kind: toolkit.SourceKindRepo}
	router := skn.NewRouter(skn.WithGitDriver(driver))
	mux := newHarvesterServeMux(router)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("readyz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
