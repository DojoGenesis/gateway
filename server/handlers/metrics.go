package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

type MetricsResponse struct {
	Cache       CacheMetrics       `json:"cache"`
	System      SystemMetrics      `json:"system"`
	Performance PerformanceMetrics `json:"performance,omitempty"`
	Timestamp   time.Time          `json:"timestamp"`
}

type CacheMetrics struct {
	Enabled   bool    `json:"enabled"`
	Size      int     `json:"size"`
	MaxSize   int     `json:"max_size"`
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	TotalReqs int64   `json:"total_requests"`
}

type SystemMetrics struct {
	Goroutines int     `json:"goroutines"`
	MemoryMB   float64 `json:"memory_mb"`
	AllocMB    float64 `json:"alloc_mb"`
	NumGC      uint32  `json:"num_gc"`
}

type PerformanceMetrics struct {
	SimpleQueryAvgMs  float64 `json:"simple_query_avg_ms,omitempty"`
	ComplexQueryAvgMs float64 `json:"complex_query_avg_ms,omitempty"`
}

// MetricsHandler handles metrics HTTP requests.
type MetricsHandler struct {
	chatHandler *ChatHandler
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(ch *ChatHandler) *MetricsHandler {
	return &MetricsHandler{chatHandler: ch}
}

func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	var cacheMetrics CacheMetrics

	if h.chatHandler != nil && h.chatHandler.cache != nil {
		hits, misses, hitRate := h.chatHandler.cache.Stats()
		cacheMetrics = CacheMetrics{
			Enabled:   true,
			Size:      h.chatHandler.cache.Size(),
			MaxSize:   1000,
			Hits:      hits,
			Misses:    misses,
			HitRate:   hitRate,
			TotalReqs: hits + misses,
		}
	} else {
		cacheMetrics = CacheMetrics{
			Enabled: false,
		}
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	systemMetrics := SystemMetrics{
		Goroutines: runtime.NumGoroutine(),
		MemoryMB:   float64(m.Sys) / 1024 / 1024,
		AllocMB:    float64(m.Alloc) / 1024 / 1024,
		NumGC:      m.NumGC,
	}

	response := MetricsResponse{
		Cache:     cacheMetrics,
		System:    systemMetrics,
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

func (h *MetricsHandler) ClearCache(c *gin.Context) {
	if h.chatHandler == nil || h.chatHandler.cache == nil {
		respondError(c, http.StatusServiceUnavailable, "cache not initialized")
		return
	}

	h.chatHandler.cache.Clear()

	c.JSON(http.StatusOK, gin.H{
		"message":   "cache cleared successfully",
		"timestamp": time.Now(),
	})
}

func (h *MetricsHandler) DisableCache(c *gin.Context) {
	if h.chatHandler == nil || h.chatHandler.cache == nil {
		respondError(c, http.StatusServiceUnavailable, "cache not initialized")
		return
	}

	h.chatHandler.cache.Disable()

	c.JSON(http.StatusOK, gin.H{
		"message":   "cache disabled",
		"timestamp": time.Now(),
	})
}

func (h *MetricsHandler) EnableCache(c *gin.Context) {
	if h.chatHandler == nil || h.chatHandler.cache == nil {
		respondError(c, http.StatusServiceUnavailable, "cache not initialized")
		return
	}

	h.chatHandler.cache.Enable()

	c.JSON(http.StatusOK, gin.H{
		"message":   "cache enabled",
		"timestamp": time.Now(),
	})
}
