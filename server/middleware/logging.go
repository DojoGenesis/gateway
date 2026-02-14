package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		if raw != "" {
			path = path + "?" + raw
		}

		attrs := []any{
			"method", method,
			"path", path,
			"status", statusCode,
			"latency", latency,
			"client_ip", clientIP,
		}

		if rid, exists := c.Get("request_id"); exists {
			attrs = append(attrs, "request_id", rid)
		}

		if errStr := c.Errors.String(); errStr != "" {
			attrs = append(attrs, "errors", errStr)
		}

		if statusCode >= 500 {
			slog.Error("request completed", attrs...)
		} else if statusCode >= 400 {
			slog.Warn("request completed", attrs...)
		} else {
			slog.Info("request completed", attrs...)
		}
	}
}
