package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// ServeStdio reads JSON-RPC requests from reader and writes responses to writer.
// It processes one request per line (newline-delimited JSON-RPC).
func (s *mcpServer) ServeStdio(ctx context.Context, reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	// Increase buffer for large messages.
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				Error: &jsonRPCError{
					Code:    -32700,
					Message: "parse error",
				},
			}
			writeResponse(writer, resp)
			continue
		}

		if req.JSONRPC != "2.0" {
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &jsonRPCError{
					Code:    -32600,
					Message: "invalid request: jsonrpc must be 2.0",
				},
			}
			writeResponse(writer, resp)
			continue
		}

		result, err := s.HandleMessage(ctx, req.Method, req.Params)
		if err != nil {
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &jsonRPCError{
					Code:    -32603,
					Message: err.Error(),
				},
			}
			writeResponse(writer, resp)
			continue
		}

		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}
		writeResponse(writer, resp)
	}

	return scanner.Err()
}

func writeResponse(w io.Writer, resp jsonRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		// Last-resort fallback: write a minimal error response.
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","error":{"code":-32603,"message":"internal: marshal error"}}`+"\n")
		return
	}
	_, _ = fmt.Fprintln(w, string(data))
}
