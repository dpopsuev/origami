package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"go.lsp.dev/jsonrpc2"
)

func TestNewStdioStream(t *testing.T) {
	r := &bytes.Buffer{}
	w := &bytes.Buffer{}

	stream := NewStdioStream(r, w)
	if stream == nil {
		t.Fatal("NewStdioStream returned nil")
	}

	var _ jsonrpc2.Stream = stream
}

func TestServeStream(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	stream := NewStdioStream(serverR, serverW)
	srv := NewServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := ServeStream(ctx, srv, stream)
	if conn == nil {
		t.Fatal("ServeStream returned nil conn")
	}

	clientStream := NewStdioStream(clientR, clientW)
	clientConn := jsonrpc2.NewConn(clientStream)
	clientConn.Go(ctx, jsonrpc2.MethodNotFoundHandler)

	var result json.RawMessage
	_, err := clientConn.Call(ctx, "initialize", json.RawMessage(`{
		"processId": null,
		"capabilities": {},
		"rootUri": "file:///tmp/test"
	}`), &result)
	if err != nil {
		t.Fatalf("initialize call: %v", err)
	}
	if len(result) == 0 {
		t.Error("empty initialize response")
	}

	cancel()
}
