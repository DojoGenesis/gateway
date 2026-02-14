package tools

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkAllTools benchmarks all 27 tools
func BenchmarkAllTools(b *testing.B) {
	ctx := context.Background()

	// Setup temporary directories
	tmpDir, err := ioutil.TempDir("", "benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := ioutil.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		b.Fatal(err)
	}

	// Create test HTML server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test</h1><a href='/link1'>Link 1</a></body></html>"))
	}))
	defer testServer.Close()

	b.Run("FileOperations", func(b *testing.B) {
		benchmarkFileOperations(b, ctx, tmpDir, testFile)
	})

	b.Run("WebOperations", func(b *testing.B) {
		benchmarkWebOperations(b, ctx, testServer.URL)
	})

	b.Run("Computation", func(b *testing.B) {
		benchmarkComputation(b, ctx)
	})

	b.Run("System", func(b *testing.B) {
		benchmarkSystem(b, ctx)
	})

	b.Run("Planning", func(b *testing.B) {
		benchmarkPlanning(b, ctx, tmpDir)
	})

	b.Run("Research", func(b *testing.B) {
		benchmarkResearch(b, ctx)
	})

	b.Run("Meta", func(b *testing.B) {
		benchmarkMeta(b, ctx)
	})
}

func benchmarkFileOperations(b *testing.B, ctx context.Context, tmpDir, testFile string) {
	b.Run("ReadFile", func(b *testing.B) {
		params := map[string]interface{}{
			"file_path": testFile,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := ReadFile(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WriteFile", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			params := map[string]interface{}{
				"file_path": filepath.Join(tmpDir, fmt.Sprintf("write_%d.txt", i)),
				"content":   "benchmark content",
			}
			_, err := WriteFile(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ListDirectory", func(b *testing.B) {
		params := map[string]interface{}{
			"directory_path": tmpDir,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := ListDirectory(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SearchFiles", func(b *testing.B) {
		params := map[string]interface{}{
			"root_path": tmpDir,
			"pattern":   "*.txt",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SearchFiles(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchmarkWebOperations(b *testing.B, ctx context.Context, testURL string) {
	b.Run("FetchURL", func(b *testing.B) {
		params := map[string]interface{}{
			"url": testURL,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := FetchURL(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("APICall", func(b *testing.B) {
		params := map[string]interface{}{
			"url":    testURL,
			"method": "GET",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := APICall(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WebExtractLinks", func(b *testing.B) {
		params := map[string]interface{}{
			"url": testURL,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := WebExtractLinks(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchmarkComputation(b *testing.B, ctx context.Context) {
	b.Run("Calculate", func(b *testing.B) {
		params := map[string]interface{}{
			"expression": "(2 + 2) * 10 / 4",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Calculate(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchmarkSystem(b *testing.B, ctx context.Context) {
	b.Run("RunCommand", func(b *testing.B) {
		params := map[string]interface{}{
			"command": "echo 'hello'",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := RunCommand(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchmarkPlanning(b *testing.B, ctx context.Context, tmpDir string) {
	planDir := filepath.Join(tmpDir, ".dojo", "planning")
	os.MkdirAll(planDir, 0755)
	os.Setenv("DOJO_PLANNING_DIR", planDir)
	defer os.Unsetenv("DOJO_PLANNING_DIR")

	b.Run("CreatePlan", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			params := map[string]interface{}{
				"title":       fmt.Sprintf("Benchmark Plan %d", i),
				"description": "Test plan",
				"goals":       []interface{}{"Goal 1", "Goal 2"},
			}
			_, err := CreatePlan(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Create a test plan first
	createParams := map[string]interface{}{
		"title":       "Test Plan",
		"description": "Test plan",
		"goals":       []interface{}{"Goal 1"},
	}
	result, _ := CreatePlan(ctx, createParams)
	planID := result["plan"].(map[string]interface{})["id"].(string)

	b.Run("UpdatePlan", func(b *testing.B) {
		params := map[string]interface{}{
			"plan_id":     planID,
			"title":       "Updated Plan",
			"description": "Updated description",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := UpdatePlan(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("TrackProgress", func(b *testing.B) {
		params := map[string]interface{}{
			"plan_id": planID,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := TrackProgress(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ValidatePlan", func(b *testing.B) {
		params := map[string]interface{}{
			"plan_id": planID,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := ValidatePlan(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CreateMilestone", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			params := map[string]interface{}{
				"plan_id":     planID,
				"title":       fmt.Sprintf("Milestone %d", i),
				"description": "Test milestone",
				"due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			}
			_, err := CreateMilestone(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchmarkResearch(b *testing.B, ctx context.Context) {
	b.Run("SynthesizeInfo", func(b *testing.B) {
		params := map[string]interface{}{
			"sources": []interface{}{
				map[string]interface{}{
					"content": "The sky is blue.",
					"url":     "https://example.com/1",
				},
				map[string]interface{}{
					"content": "Water is wet.",
					"url":     "https://example.com/2",
				},
			},
			"topic": "Nature",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SynthesizeInfo(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("AnalyzeSentiment", func(b *testing.B) {
		params := map[string]interface{}{
			"text": "This is a wonderful day! I love it!",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := AnalyzeSentiment(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ExtractEntities", func(b *testing.B) {
		params := map[string]interface{}{
			"text": "Apple Inc. is located in Cupertino, California. Tim Cook is the CEO.",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := ExtractEntities(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchmarkMeta(b *testing.B, ctx context.Context) {
	b.Run("SearchTools", func(b *testing.B) {
		params := map[string]interface{}{
			"query": "file",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SearchTools(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetToolInfo", func(b *testing.B) {
		params := map[string]interface{}{
			"tool_name": "read_file",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := GetToolInfo(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkToolRegistry benchmarks registry operations
func BenchmarkToolRegistry(b *testing.B) {
	b.Run("GetTool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := GetTool("read_file")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetAllTools", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tools := GetAllTools()
			if len(tools) == 0 {
				b.Fatal("no tools returned")
			}
		}
	})

	b.Run("InvokeTool", func(b *testing.B) {
		ctx := context.Background()
		params := map[string]interface{}{
			"expression": "2 + 2",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := InvokeTool(ctx, "calculate", params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentToolCalls benchmarks parallel tool execution
func BenchmarkConcurrentToolCalls(b *testing.B) {
	ctx := context.Background()

	benchmarks := []struct {
		name       string
		concurrent int
	}{
		{"Concurrent_10", 10},
		{"Concurrent_100", 100},
		{"Concurrent_1000", 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				done := make(chan bool, bm.concurrent)
				for j := 0; j < bm.concurrent; j++ {
					go func() {
						params := map[string]interface{}{
							"expression": "2 + 2",
						}
						_, _ = Calculate(ctx, params)
						done <- true
					}()
				}
				for j := 0; j < bm.concurrent; j++ {
					<-done
				}
			}
		})
	}
}
