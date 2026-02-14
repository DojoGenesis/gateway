package provider

import (
	"context"
	"encoding/json"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider/pb"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type ModelProviderGRPCPlugin struct {
	plugin.Plugin
	Impl ModelProvider
}

func (p *ModelProviderGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterModelProviderServer(s, &ModelProviderGRPCServer{
		Impl:   p.Impl,
		broker: broker,
	})
	return nil
}

func (p *ModelProviderGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &ModelProviderGRPCClient{
		client: pb.NewModelProviderClient(c),
		broker: broker,
	}, nil
}

type ModelProviderGRPCServer struct {
	pb.UnimplementedModelProviderServer
	Impl   ModelProvider
	broker *plugin.GRPCBroker
}

func (s *ModelProviderGRPCServer) GetInfo(ctx context.Context, req *pb.GetInfoRequest) (*pb.ProviderInfo, error) {
	info, err := s.Impl.GetInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.ProviderInfo{
		Name:         info.Name,
		Version:      info.Version,
		Description:  info.Description,
		Capabilities: info.Capabilities,
	}, nil
}

func (s *ModelProviderGRPCServer) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	models, err := s.Impl.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.ListModelsResponse{Models: convertModelInfoSlice(models)}, nil
}

func (s *ModelProviderGRPCServer) GenerateCompletion(ctx context.Context, req *pb.CompletionRequest) (*pb.CompletionResponse, error) {
	pluginReq := convertFromProtoCompletionRequest(req)
	resp, err := s.Impl.GenerateCompletion(ctx, pluginReq)
	if err != nil {
		return nil, err
	}
	return convertToProtoCompletionResponse(resp), nil
}

func (s *ModelProviderGRPCServer) GenerateCompletionStream(req *pb.CompletionRequest, stream pb.ModelProvider_GenerateCompletionStreamServer) error {
	pluginReq := convertFromProtoCompletionRequest(req)
	chunkCh, err := s.Impl.GenerateCompletionStream(stream.Context(), pluginReq)
	if err != nil {
		return err
	}

	for chunk := range chunkCh {
		protoChunk := &pb.CompletionChunk{
			Id:    chunk.ID,
			Delta: chunk.Delta,
			Done:  chunk.Done,
		}
		if err := stream.Send(protoChunk); err != nil {
			return err
		}
	}

	return nil
}

func (s *ModelProviderGRPCServer) CallTool(ctx context.Context, req *pb.ToolCallRequest) (*pb.ToolCallResponse, error) {
	var toolCallArgs map[string]interface{}
	if err := json.Unmarshal(req.ToolCall.Arguments, &toolCallArgs); err != nil {
		return &pb.ToolCallResponse{Error: "invalid tool call arguments"}, nil
	}

	var contextMap map[string]interface{}
	if len(req.Context) > 0 {
		if err := json.Unmarshal(req.Context, &contextMap); err != nil {
			return &pb.ToolCallResponse{Error: "invalid context"}, nil
		}
	}

	pluginReq := &ToolCallRequest{
		ToolCall: ToolCall{
			ID:        req.ToolCall.Id,
			Name:      req.ToolCall.Name,
			Arguments: toolCallArgs,
		},
		Context: contextMap,
	}

	resp, err := s.Impl.CallTool(ctx, pluginReq)
	if err != nil {
		return &pb.ToolCallResponse{Error: err.Error()}, nil
	}

	resultBytes, _ := json.Marshal(resp.Result)
	return &pb.ToolCallResponse{
		Result: resultBytes,
		Error:  resp.Error,
	}, nil
}

func (s *ModelProviderGRPCServer) GenerateEmbedding(ctx context.Context, req *pb.EmbeddingRequest) (*pb.EmbeddingResponse, error) {
	embedding, err := s.Impl.GenerateEmbedding(ctx, req.Text)
	if err != nil {
		return nil, err
	}
	return &pb.EmbeddingResponse{
		Embedding: embedding,
	}, nil
}

type ModelProviderGRPCClient struct {
	client pb.ModelProviderClient
	broker *plugin.GRPCBroker
}

func (c *ModelProviderGRPCClient) GetInfo(ctx context.Context) (*ProviderInfo, error) {
	protoInfo, err := c.client.GetInfo(ctx, &pb.GetInfoRequest{})
	if err != nil {
		return nil, err
	}
	return &ProviderInfo{
		Name:         protoInfo.Name,
		Version:      protoInfo.Version,
		Description:  protoInfo.Description,
		Capabilities: protoInfo.Capabilities,
	}, nil
}

func (c *ModelProviderGRPCClient) ListModels(ctx context.Context) ([]ModelInfo, error) {
	resp, err := c.client.ListModels(ctx, &pb.ListModelsRequest{})
	if err != nil {
		return nil, err
	}
	return convertFromProtoModelInfoSlice(resp.Models), nil
}

func (c *ModelProviderGRPCClient) GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	protoReq := convertToProtoCompletionRequest(req)
	protoResp, err := c.client.GenerateCompletion(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return convertFromProtoCompletionResponse(protoResp), nil
}

func (c *ModelProviderGRPCClient) GenerateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan *CompletionChunk, error) {
	protoReq := convertToProtoCompletionRequest(req)
	stream, err := c.client.GenerateCompletionStream(ctx, protoReq)
	if err != nil {
		return nil, err
	}

	ch := make(chan *CompletionChunk, 10)

	go func() {
		defer close(ch)

		for {
			protoChunk, err := stream.Recv()
			if err != nil {
				return
			}

			chunk := &CompletionChunk{
				ID:    protoChunk.Id,
				Delta: protoChunk.Delta,
				Done:  protoChunk.Done,
			}
			ch <- chunk

			if chunk.Done {
				return
			}
		}
	}()

	return ch, nil
}

func (c *ModelProviderGRPCClient) CallTool(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
	argsBytes, _ := json.Marshal(req.ToolCall.Arguments)
	contextBytes, _ := json.Marshal(req.Context)

	protoReq := &pb.ToolCallRequest{
		ToolCall: &pb.ToolCall{
			Id:        req.ToolCall.ID,
			Name:      req.ToolCall.Name,
			Arguments: argsBytes,
		},
		Context: contextBytes,
	}

	protoResp, err := c.client.CallTool(ctx, protoReq)
	if err != nil {
		return &ToolCallResponse{Error: err.Error()}, nil
	}

	var result interface{}
	if len(protoResp.Result) > 0 {
		json.Unmarshal(protoResp.Result, &result)
	}

	return &ToolCallResponse{
		Result: result,
		Error:  protoResp.Error,
	}, nil
}

func (c *ModelProviderGRPCClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	protoReq := &pb.EmbeddingRequest{
		Text: text,
	}
	protoResp, err := c.client.GenerateEmbedding(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return protoResp.Embedding, nil
}

func convertModelInfoSlice(models []ModelInfo) []*pb.ModelInfo {
	result := make([]*pb.ModelInfo, len(models))
	for i, m := range models {
		result[i] = &pb.ModelInfo{
			Id:          m.ID,
			Name:        m.Name,
			Provider:    m.Provider,
			ContextSize: int32(m.ContextSize),
			Cost:        m.Cost,
		}
	}
	return result
}

func convertFromProtoModelInfoSlice(protoModels []*pb.ModelInfo) []ModelInfo {
	result := make([]ModelInfo, len(protoModels))
	for i, m := range protoModels {
		result[i] = ModelInfo{
			ID:          m.Id,
			Name:        m.Name,
			Provider:    m.Provider,
			ContextSize: int(m.ContextSize),
			Cost:        m.Cost,
		}
	}
	return result
}

func convertToProtoCompletionRequest(req *CompletionRequest) *pb.CompletionRequest {
	messages := make([]*pb.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = &pb.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	tools := make([]*pb.Tool, len(req.Tools))
	for i, t := range req.Tools {
		paramsBytes, _ := json.Marshal(t.Parameters)
		tools[i] = &pb.Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  paramsBytes,
		}
	}

	return &pb.CompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   int32(req.MaxTokens),
		Tools:       tools,
		Stream:      req.Stream,
	}
}

func convertFromProtoCompletionRequest(protoReq *pb.CompletionRequest) *CompletionRequest {
	messages := make([]Message, len(protoReq.Messages))
	for i, m := range protoReq.Messages {
		messages[i] = Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	tools := make([]Tool, len(protoReq.Tools))
	for i, t := range protoReq.Tools {
		var params map[string]interface{}
		json.Unmarshal(t.Parameters, &params)
		tools[i] = Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
		}
	}

	return &CompletionRequest{
		Model:       protoReq.Model,
		Messages:    messages,
		Temperature: protoReq.Temperature,
		MaxTokens:   int(protoReq.MaxTokens),
		Tools:       tools,
		Stream:      protoReq.Stream,
	}
}

func convertToProtoCompletionResponse(resp *CompletionResponse) *pb.CompletionResponse {
	toolCalls := make([]*pb.ToolCall, len(resp.ToolCalls))
	for i, tc := range resp.ToolCalls {
		argsBytes, _ := json.Marshal(tc.Arguments)
		toolCalls[i] = &pb.ToolCall{
			Id:        tc.ID,
			Name:      tc.Name,
			Arguments: argsBytes,
		}
	}

	return &pb.CompletionResponse{
		Id:      resp.ID,
		Model:   resp.Model,
		Content: resp.Content,
		Usage: &pb.Usage{
			InputTokens:  int32(resp.Usage.InputTokens),
			OutputTokens: int32(resp.Usage.OutputTokens),
			TotalTokens:  int32(resp.Usage.TotalTokens),
		},
		ToolCalls: toolCalls,
	}
}

func convertFromProtoCompletionResponse(protoResp *pb.CompletionResponse) *CompletionResponse {
	toolCalls := make([]ToolCall, len(protoResp.ToolCalls))
	for i, tc := range protoResp.ToolCalls {
		var args map[string]interface{}
		json.Unmarshal(tc.Arguments, &args)
		toolCalls[i] = ToolCall{
			ID:        tc.Id,
			Name:      tc.Name,
			Arguments: args,
		}
	}

	return &CompletionResponse{
		ID:      protoResp.Id,
		Model:   protoResp.Model,
		Content: protoResp.Content,
		Usage: Usage{
			InputTokens:  int(protoResp.Usage.InputTokens),
			OutputTokens: int(protoResp.Usage.OutputTokens),
			TotalTokens:  int(protoResp.Usage.TotalTokens),
		},
		ToolCalls: toolCalls,
	}
}
