package provider

import (
	"context"
	"encoding/gob"
	"io"
	"net/rpc"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/go-plugin"
)

var (
	rpcMetadataTimeout    = getEnvDuration("RPC_METADATA_TIMEOUT", 30*time.Second)
	rpcCompletionTimeout  = getEnvDuration("RPC_COMPLETION_TIMEOUT", 5*time.Minute)
	rpcStreamTimeout      = getEnvDuration("RPC_STREAM_TIMEOUT", 10*time.Minute)
	rpcToolCallTimeout    = getEnvDuration("RPC_TOOL_CALL_TIMEOUT", 2*time.Minute)
)

func getEnvDuration(envKey string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(envKey); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultVal
}

/*
RPC Implementation Notes:

1. Protocol: NetRPC with MuxBroker
   - Uses NetRPC for RPC calls (simple, no protobuf dependencies)
   - Uses MuxBroker for bidirectional streaming (GenerateCompletionStream)
   - Production alternative: GRPC provides native streaming and context support

2. Context Propagation:
   - Client: Uses async RPC (client.Go) with select pattern for cancellation
   - Server: Uses WithTimeout for operations (30s metadata, 5-10min completions)
   - Streaming: Context cancellation closes the stream channel

3. Streaming:
   - Implemented via MuxBroker (multiplexed streams over single RPC connection)
   - Server creates stream ID, client dials to connect
   - Uses gob encoding for chunk serialization
   - Proper cleanup on context cancellation or stream completion

4. Thread Safety:
   - All RPC methods are thread-safe (go-plugin handles concurrency)
   - MuxBroker handles multiple concurrent streams
*/

type ModelProviderPlugin struct {
	Impl ModelProvider
}

func (p *ModelProviderPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &ModelProviderRPCServer{Impl: p.Impl, broker: b}, nil
}

func (p *ModelProviderPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ModelProviderRPCClient{client: c, broker: b}, nil
}

type ModelProviderRPCServer struct {
	Impl   ModelProvider
	broker *plugin.MuxBroker
}

func (s *ModelProviderRPCServer) GetInfo(args interface{}, resp *ProviderInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcMetadataTimeout)
	defer cancel()

	info, err := s.Impl.GetInfo(ctx)
	if err != nil {
		return err
	}
	*resp = *info
	return nil
}

func (s *ModelProviderRPCServer) ListModels(args interface{}, resp *[]ModelInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcMetadataTimeout)
	defer cancel()

	models, err := s.Impl.ListModels(ctx)
	if err != nil {
		return err
	}
	*resp = models
	return nil
}

func (s *ModelProviderRPCServer) GenerateCompletion(req *CompletionRequest, resp *CompletionResponse) error {
	timeout := rpcCompletionTimeout
	if req.MaxTokens > 0 {
		tokenTimeout := time.Duration(req.MaxTokens/10) * time.Second
		if tokenTimeout > timeout {
			timeout = tokenTimeout
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	completion, err := s.Impl.GenerateCompletion(ctx, req)
	if err != nil {
		return err
	}
	*resp = *completion
	return nil
}

type StreamRequest struct {
	CompletionRequest *CompletionRequest
	BrokerID          uint32
}

type StreamResponse struct {
	BrokerID uint32
}

func (s *ModelProviderRPCServer) GenerateCompletionStream(req *StreamRequest, resp *StreamResponse) error {
	brokerID := s.broker.NextId()
	resp.BrokerID = brokerID

	go func() {
		conn, err := s.broker.Accept(brokerID)
		if err != nil {
			return
		}
		defer conn.Close()

		encoder := gob.NewEncoder(conn)

		timeout := rpcStreamTimeout
		if req.CompletionRequest.MaxTokens > 0 {
			tokenTimeout := time.Duration(req.CompletionRequest.MaxTokens/5) * time.Second
			if tokenTimeout > timeout {
				timeout = tokenTimeout
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		chunkCh, err := s.Impl.GenerateCompletionStream(ctx, req.CompletionRequest)
		if err != nil {
			encoder.Encode(&CompletionChunk{Done: true})
			return
		}

		for chunk := range chunkCh {
			if err := encoder.Encode(chunk); err != nil {
				return
			}
		}
	}()

	return nil
}

func (s *ModelProviderRPCServer) CallTool(req *ToolCallRequest, resp *ToolCallResponse) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcToolCallTimeout)
	defer cancel()

	result, err := s.Impl.CallTool(ctx, req)
	if err != nil {
		return err
	}
	*resp = *result
	return nil
}

type ModelProviderRPCClient struct {
	client *rpc.Client
	broker *plugin.MuxBroker
}

func (c *ModelProviderRPCClient) GetInfo(ctx context.Context) (*ProviderInfo, error) {
	var resp ProviderInfo
	call := c.client.Go("Plugin.GetInfo", new(interface{}), &resp, nil)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.Done:
		if call.Error != nil {
			return nil, call.Error
		}
		return &resp, nil
	}
}

func (c *ModelProviderRPCClient) ListModels(ctx context.Context) ([]ModelInfo, error) {
	var resp []ModelInfo
	call := c.client.Go("Plugin.ListModels", new(interface{}), &resp, nil)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.Done:
		if call.Error != nil {
			return nil, call.Error
		}
		return resp, nil
	}
}

func (c *ModelProviderRPCClient) GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	var resp CompletionResponse
	call := c.client.Go("Plugin.GenerateCompletion", req, &resp, nil)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.Done:
		if call.Error != nil {
			return nil, call.Error
		}
		return &resp, nil
	}
}

func (c *ModelProviderRPCClient) GenerateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan *CompletionChunk, error) {
	var resp StreamResponse
	streamReq := &StreamRequest{
		CompletionRequest: req,
	}

	call := c.client.Go("Plugin.GenerateCompletionStream", streamReq, &resp, nil)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.Done:
		if call.Error != nil {
			return nil, call.Error
		}
	}

	conn, err := c.broker.Dial(resp.BrokerID)
	if err != nil {
		return nil, err
	}

	ch := make(chan *CompletionChunk, 10)

	go func() {
		defer close(ch)
		defer conn.Close()

		decoder := gob.NewDecoder(conn)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				var chunk CompletionChunk
				if err := decoder.Decode(&chunk); err != nil {
					if err != io.EOF {
						ch <- &CompletionChunk{Done: true}
					}
					return
				}

				ch <- &chunk
				if chunk.Done {
					return
				}
			}
		}
	}()

	return ch, nil
}

func (c *ModelProviderRPCClient) CallTool(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
	var resp ToolCallResponse
	call := c.client.Go("Plugin.CallTool", req, &resp, nil)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.Done:
		if call.Error != nil {
			return nil, call.Error
		}
		return &resp, nil
	}
}
