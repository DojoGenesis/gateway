package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

type BenchmarkMockPluginManager struct{}

func (m *BenchmarkMockPluginManager) GetProvider(name string) (providerpkg.ModelProvider, error) {
	return &BenchmarkMockProvider{}, nil
}

func (m *BenchmarkMockPluginManager) GetProviders() map[string]providerpkg.ModelProvider {
	return map[string]providerpkg.ModelProvider{
		"mock": &BenchmarkMockProvider{},
	}
}

type BenchmarkMockProvider struct{}

func (mp *BenchmarkMockProvider) GetInfo(ctx context.Context) (*providerpkg.ProviderInfo, error) {
	return &providerpkg.ProviderInfo{
		Name:        "mock",
		Version:     "1.0.0",
		Description: "Mock provider for benchmarking",
	}, nil
}

func (mp *BenchmarkMockProvider) ListModels(ctx context.Context) ([]providerpkg.ModelInfo, error) {
	return []providerpkg.ModelInfo{
		{
			ID:   "mock-model",
			Name: "Mock Model",
		},
	}, nil
}

func (mp *BenchmarkMockProvider) GenerateCompletion(ctx context.Context, req *providerpkg.CompletionRequest) (*providerpkg.CompletionResponse, error) {
	return &providerpkg.CompletionResponse{
		ID:      "mock-response",
		Content: "This is a mock response for benchmarking purposes.",
		Model:   req.Model,
		Usage: providerpkg.Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
		ToolCalls: []providerpkg.ToolCall{},
	}, nil
}

func (mp *BenchmarkMockProvider) GenerateCompletionStream(ctx context.Context, req *providerpkg.CompletionRequest) (<-chan *providerpkg.CompletionChunk, error) {
	ch := make(chan *providerpkg.CompletionChunk, 1)
	go func() {
		defer close(ch)
		ch <- &providerpkg.CompletionChunk{
			ID:    "mock-chunk",
			Delta: "mock",
			Done:  true,
		}
	}()
	return ch, nil
}

func (mp *BenchmarkMockProvider) CallTool(ctx context.Context, req *providerpkg.ToolCallRequest) (*providerpkg.ToolCallResponse, error) {
	return &providerpkg.ToolCallResponse{
		Result: map[string]interface{}{"result": "mock"},
		Error:  "",
	}, nil
}

func (mp *BenchmarkMockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

func BenchmarkMiniDelegationAgent(b *testing.B) {
	agent := NewMiniDelegationAgent()

	queries := []string{
		"Analyze the performance of this code",
		"Search for information about quantum computing",
		"Build a web scraper in Python",
		"Debug this error in my application",
		"What is the meaning of life?",
	}

	b.Run("ClassifyIntent", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_, _ = agent.ClassifyIntent(ctx, query)
		}
	})

	b.Run("ClassifyIntent", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_, _ = agent.ClassifyIntent(ctx, query)
		}
	})
}

func BenchmarkPrimaryAgent(b *testing.B) {
	tmpFile, err := ioutil.TempFile("", "bench_agent_memory_*.db")
	if err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	memMgr, err := memory.NewMemoryManager(tmpFile.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer memMgr.Close()

	pluginMgr := &BenchmarkMockPluginManager{}

	agent := NewPrimaryAgentWithConfig(pluginMgr, "mock", "mock", "mock")
	agent.SetMemoryManager(memMgr)
	agent.SetTimeout(30 * time.Second)

	ctx := context.Background()

	b.Run("HandleQuery", func(b *testing.B) {
		query := "What is the capital of France?"
		providerName := "mock"
		modelID := "mock-model"
		userID := "test-user"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := agent.HandleQuery(ctx, query, providerName, modelID, userID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("HandleQueryWithContext", func(b *testing.B) {
		// Store some context
		for i := 0; i < 5; i++ {
			mem := memory.Memory{
				ID:      fmt.Sprintf("ctx-%d", i),
				Type:    "conversation",
				Content: fmt.Sprintf("Previous message %d", i),
				Metadata: map[string]interface{}{
					"user_id": "user1",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			memMgr.Store(ctx, mem)
		}

		query := "Tell me more about that"
		providerName := "mock"
		modelID := "mock-model"
		userID := "user1"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := agent.HandleQuery(ctx, query, providerName, modelID, userID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("IntentClassificationOverhead", func(b *testing.B) {
		query := "Analyze this code for bugs"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = agent.miniAgent.ClassifyIntent(ctx, query)
		}
	})
}

func BenchmarkConcurrentAgentRequests(b *testing.B) {
	tmpFile, err := ioutil.TempFile("", "bench_concurrent_agent_*.db")
	if err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	memMgr, err := memory.NewMemoryManager(tmpFile.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer memMgr.Close()

	pluginMgr := &BenchmarkMockPluginManager{}

	agent := NewPrimaryAgentWithConfig(pluginMgr, "mock", "mock", "mock")
	agent.SetMemoryManager(memMgr)
	agent.SetTimeout(30 * time.Second)

	ctx := context.Background()

	benchmarks := []struct {
		name       string
		concurrent int
	}{
		{"Concurrent_10", 10},
		{"Concurrent_100", 100},
		{"Concurrent_1000", 1000},
		{"Concurrent_10000", 10000},
	}

	queries := []string{
		"What is AI?",
		"Explain quantum computing",
		"How does blockchain work?",
		"What is machine learning?",
		"Describe neural networks",
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

						query := queries[idx%len(queries)]
						providerName := "mock"
						modelID := "mock-model"
						userID := fmt.Sprintf("user-%d", idx)

						agent.HandleQuery(ctx, query, providerName, modelID, userID)
					}(j)
				}

				wg.Wait()
			}
		})
	}
}

func BenchmarkAgentWithMemory(b *testing.B) {
	tmpFile, err := ioutil.TempFile("", "bench_agent_with_memory_*.db")
	if err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	memMgr, err := memory.NewMemoryManager(tmpFile.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer memMgr.Close()

	pluginMgr := &BenchmarkMockPluginManager{}

	agent := NewPrimaryAgentWithConfig(pluginMgr, "mock", "mock", "mock")
	agent.SetMemoryManager(memMgr)
	agent.SetTimeout(30 * time.Second)

	ctx := context.Background()

	memorySizes := []int{0, 10, 100, 1000}

	for _, size := range memorySizes {
		b.Run(fmt.Sprintf("MemorySize_%d", size), func(b *testing.B) {
			// Pre-populate memory
			for i := 0; i < size; i++ {
				mem := memory.Memory{
					ID:      fmt.Sprintf("mem-%d", i),
					Type:    "conversation",
					Content: fmt.Sprintf("Previous conversation %d", i),
					Metadata: map[string]interface{}{
						"user_id": "user1",
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				memMgr.Store(ctx, mem)
			}

			query := "Continue our conversation"
			providerName := "mock"
			modelID := "mock-model"
			userID := "user1"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := agent.HandleQuery(ctx, query, providerName, modelID, userID)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkProviderSelection(b *testing.B) {
	agent := &PrimaryAgent{
		miniAgent: NewMiniDelegationAgent(),
	}

	ctx := context.Background()

	b.Run("SelectProvider", func(b *testing.B) {
		queries := []string{
			"Think about this problem",
			"Search for information",
			"Build something",
			"Debug this code",
			"General question",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			intent, _ := agent.miniAgent.ClassifyIntent(ctx, query)
			_ = agent.selectProvider("authenticated", intent)
		}
	})
}
