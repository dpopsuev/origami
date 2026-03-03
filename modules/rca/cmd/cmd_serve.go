package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/dpopsuev/origami/kami"
	"github.com/dpopsuev/origami/logging"
	mcpserver "github.com/dpopsuev/origami/modules/rca/mcpconfig"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

var kamiPort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server over stdio for Cursor integration",
	Long: `Starts an MCP server over stdin/stdout. Cursor connects via .cursor/mcp.json
and calls calibration tools directly — no file-based signal protocol needed.

The server monitors for parent process death. When Cursor disconnects or
restarts its extension host, the server self-terminates to prevent zombie processes.`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().IntVar(&kamiPort, "kami-port", 3001,
		"Port for the Kami SSE server (live circuit visualization). Set 0 to disable.")
}

func runServe(cmd *cobra.Command, _ []string) error {
	log := logging.New("mcp")

	srv := mcpserver.NewServer()
	defer srv.Shutdown()

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	if kamiPort > 0 {
		bridge := kami.NewEventBridge(nil)
		kamiSrv := kami.NewServer(kami.Config{
			Port:   kamiPort,
			Bridge: bridge,
		})
		srv.KamiServer = kamiSrv

		kamiErr := make(chan error, 1)
		go func() {
			kamiErr <- kamiSrv.Start(ctx)
		}()

		// Start binds synchronously, so a port conflict arrives fast.
		// Give it 500ms to either fail or begin serving.
		select {
		case err := <-kamiErr:
			if err != nil && ctx.Err() == nil {
				log.Error(fmt.Sprintf("kami failed to start on port %d: %v (live visualization disabled)", kamiPort, err))
				srv.KamiServer = nil
			}
		case <-time.After(500 * time.Millisecond):
			log.Info(fmt.Sprintf("kami SSE server started, connect sumi via: origami sumi --watch 127.0.0.1:%d", kamiPort))
		}
	}

	mcpserver.WatchStdin(ctx, nil, cancel)

	log.Info("starting asterisk MCP server over stdio (parent watchdog active)")
	return srv.MCPServer.Run(ctx, &sdkmcp.StdioTransport{})
}
