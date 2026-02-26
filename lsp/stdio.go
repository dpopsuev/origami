package lsp

import (
	"context"
	"io"

	"go.lsp.dev/jsonrpc2"
)

type readWriteCloser struct {
	io.Reader
	io.Writer
}

func (rwc readWriteCloser) Close() error { return nil }

// NewStdioStream creates a JSON-RPC stream from stdin/stdout.
func NewStdioStream(r io.Reader, w io.Writer) jsonrpc2.Stream {
	return jsonrpc2.NewStream(readWriteCloser{r, w})
}

// ServeStream starts serving the LSP on the given stream.
func ServeStream(ctx context.Context, srv *Server, stream jsonrpc2.Stream) jsonrpc2.Conn {
	conn := jsonrpc2.NewConn(stream)
	conn.Go(ctx, srv.Handler())
	return conn
}
