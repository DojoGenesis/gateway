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

func HandleMetrics(c *gin.Context) {
	var cacheMetrics CacheMetrics

	if responseCache != nil {
		hits, misses, hitRate := responseCache.Stats()
		cacheMetrics = CacheMetrics{
			Enabled:   true,
			Size:      responseCache.Size(),
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

func HandleCacheClear(c *gin.Context) {
	if responseCache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache not initialized",
		})
		return
	}

	responseCache.Clear()

	c.JSON(http.StatusOK, gin.H{
		"message":   "cache cleared successfully",
		"timestamp": time.Now(),
	})
}

func HandleCacheDisable(c *gin.Context) {
	if responseCache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache not initialized",
		})
		return
	}

	responseCache.Disable()

	c.JSON(http.StatusOK, gin.H{
		"message":   "cache disabled",
		"timestamp": time.Now(),
	})
}

func HandleCacheEnable(c *gin.Context) {
	if responseCache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "cache not initialized",
		})
		return
	}

	responseCache.Enable()

	c.JSON(http.StatusOK, gin.H{
		"message":   "cache enabled",
		"timestamp": time.Now(),
	})
}
