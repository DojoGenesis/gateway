package streaming

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

type MockPluginManager struct{}

func (m *MockPluginManager) GetProvider(name string) (provider.ModelProvider, error) {
	return &MockProvider{}, nil
}

func (m *MockPluginManager) GetProviders() map[string]provider.ModelProvider {
	return map[string]provider.ModelProvider{
		"mock": &MockProvider{},
	}
}

type MockProvider struct{}

func (mp *MockProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:        "mock",
		Version:     "1.0.0",
		Description: "Mock provider for benchmarking",
	}, nil
}

func (mp *MockProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{
			ID:   "mock-model",
			Name: "Mock Model",
		},
	}, nil
}

func (mp *MockProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return &provider.CompletionResponse{
		ID:      "mock-response",
		Content: "This is a mock response for streaming benchmarks.",
		Model:   req.Model,
		Usage: provider.Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
		ToolCalls: []provider.ToolCall{},
	}, nil
}

func (mp *MockProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	ch := make(chan *provider.CompletionChunk, 1)
	go func() {
		defer close(ch)
		ch <- &provider.CompletionChunk{
			ID:    "mock-chunk",
			Delta: "mock",
			Done:  true,
		}
	}()
	return ch, nil
}

func (mp *MockProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{
		Result: map[string]interface{}{"result": "mock"},
		Error:  "",
	}, nil
}

func (mp *MockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

func setupStreamingAgent(b *testing.B) (*StreamingAgent, func()) {
	tmpFile, err := ioutil.TempFile("", "bench_streaming_*.db")
	if err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()

	memMgr, err := memory.NewMemoryManager(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		b.Fatal(err)
	}

	pluginMgr := &MockPluginManager{}

	primaryAgent := agent.NewPrimaryAgentWithConfig(pluginMgr, "mock", "mock", "mock")
	primaryAgent.SetMemoryManager(memMgr)
	primaryAgent.SetTimeout(30 * time.Second)

	streamingAgent := NewStreamingAgent(primaryAgent)

	cleanup := func() {
		memMgr.Close()
		os.Remove(tmpFile.Name())
	}

	return streamingAgent, cleanup
}

func BenchmarkStreamingAgent(b *testing.B) {
	streamingAgent, cleanup := setupStreamingAgent(b)
	defer cleanup()

	ctx := context.Background()

	b.Run("HandleQueryStreaming", func(b *testing.B) {
		req := agent.QueryRequest{
			Query:        "What is artificial intelligence?",
			UserTier:     "guest",
			ModelID:      "mock-model",
			ProviderName: "mock",
			Temperature:  0.7,
			MaxTokens:    100,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			eventChan, err := streamingAgent.HandleQueryStreaming(ctx, req)
			if err != nil {
				b.Fatal(err)
			}

			// Consume all events
			for range eventChan {
				// Process events
			}
		}
	})

	b.Run("EventGeneration", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewIntentClassifiedEvent("THINK", 0.9)
			_ = NewProviderSelectedEvent("mock", "mock-model")
			_ = NewThinkingEvent("Processing...")
			_ = NewResponseChunkEvent("chunk")
			usageMap := map[string]interface{}{
				"input_tokens":  100,
				"output_tokens": 50,
				"total_tokens":  150,
			}
			_ = NewCompleteEvent(usageMap)
		}
	})
}

func BenchmarkConcurrentStreaming(b *testing.B) {
	streamingAgent, cleanup := setupStreamingAgent(b)
	defer cleanup()

	ctx := context.Background()

	benchmarks := []struct {
		name       string
		concurrent int
	}{
		{"Concurrent_10", 10},
		{"Concurrent_100", 100},
		{"Concurrent_1000", 1000},
	}

	queries := []string{
		"Explain quantum mechanics",
		"What is blockchain?",
		"How does AI work?",
		"Describe machine learning",
		"What is cryptocurrency?",
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(bm.concurrent)

				for j := 0; j < bm.concurrent; j++ {
					go func(idx int) {
						defer wg.Done()

						req := agent.QueryRequest{
							Query:        queries[idx%len(queries)],
							UserTier:     "guest",
							ModelID:      "mock-model",
							ProviderName: "mock",
							Temperature:  0.7,
							MaxTokens:    100,
						}

						eventChan, err := streamingAgent.HandleQueryStreaming(ctx, req)
						if err != nil {
							return
						}

						// Consume events
						for range eventChan {
						}
					}(j)
				}

				wg.Wait()
			}
		})
	}
}

func BenchmarkEventChannelThroughput(b *testing.B) {
	b.Run("ChannelThroughput_100", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch := make(chan StreamEvent, 100)
			go func() {
				for j := 0; j < 100; j++ {
					ch <- NewResponseChunkEvent(fmt.Sprintf("chunk %d", j))
				}
				close(ch)
			}()
			for range ch {
			}
		}
	})

	b.Run("ChannelThroughput_1000", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch := make(chan StreamEvent, 1000)
			go func() {
				for j := 0; j < 1000; j++ {
					ch <- NewResponseChunkEvent(fmt.Sprintf("chunk %d", j))
				}
				close(ch)
			}()
			for range ch {
			}
		}
	})
}

func BenchmarkStreamEventSerialization(b *testing.B) {
	event := NewIntentClassifiedEvent("THINK", 0.9)

	b.Run("JSONMarshal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := event.ToJSON()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkStreamWithDifferentQueryTypes(b *testing.B) {
	streamingAgent, cleanup := setupStreamingAgent(b)
	defer cleanup()

	ctx := context.Background()

	queryTypes := []struct {
		name  string
		query string
	}{
		{"Simple", "Hello"},
		{"Complex", "Analyze the architectural patterns in microservices and explain how they differ from monolithic applications"},
		{"Question", "What is the capital of France?"},
		{"Command", "Write a Python function to calculate fibonacci numbers"},
		{"Multi", "First explain what AI is, then describe machine learning, and finally discuss deep learning"},
	}

	for _, qt := range queryTypes {
		b.Run(qt.name, func(b *testing.B) {
			req := agent.QueryRequest{
				Query:        qt.query,
				UserTier:     "guest",
				ModelID:      "mock-model",
				ProviderName: "mock",
				Temperature:  0.7,
				MaxTokens:    200,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				eventChan, err := streamingAgent.HandleQueryStreaming(ctx, req)
				if err != nil {
					b.Fatal(err)
				}

				eventCount := 0
				for range eventChan {
					eventCount++
				}

				if eventCount == 0 {
					b.Fatal("no events received")
				}
			}
		})
	}
}

func BenchmarkStreamingLatency(b *testing.B) {
	streamingAgent, cleanup := setupStreamingAgent(b)
	defer cleanup()

	ctx := context.Background()

	b.Run("TimeToFirstEvent", func(b *testing.B) {
		req := agent.QueryRequest{
			Query:        "Quick question",
			UserTier:     "guest",
			ModelID:      "mock-model",
			ProviderName: "mock",
			Temperature:  0.7,
			MaxTokens:    50,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()
			eventChan, err := streamingAgent.HandleQueryStreaming(ctx, req)
			if err != nil {
				b.Fatal(err)
			}

			// Wait for first event
			<-eventChan
			firstEventTime := time.Since(start)

			// Consume remaining events
			for range eventChan {
			}

			b.ReportMetric(float64(firstEventTime.Microseconds()), "μs/first-event")
		}
	})

	b.Run("TimeToCompletion", func(b *testing.B) {
		req := agent.QueryRequest{
			Query:        "Tell me about Go programming",
			UserTier:     "guest",
			ModelID:      "mock-model",
			ProviderName: "mock",
			Temperature:  0.7,
			MaxTokens:    100,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()
			eventChan, err := streamingAgent.HandleQueryStreaming(ctx, req)
			if err != nil {
				b.Fatal(err)
			}

			// Consume all events
			for range eventChan {
			}

			completionTime := time.Since(start)
			b.ReportMetric(float64(completionTime.Milliseconds()), "ms/completion")
		}
	})
}
