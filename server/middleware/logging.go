package middleware

import (
	"log"
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

		requestID := ""
		if rid, exists := c.Get("request_id"); exists {
			requestID = rid.(string)
		}

		if requestID != "" {
			log.Printf("[%s] %s %s %d %v [request_id=%s] %s",
				method,
				path,
				clientIP,
				statusCode,
				latency,
				requestID,
				c.Errors.String(),
			)
		} else {
			log.Printf("[%s] %s %s %d %v %s",
				method,
				path,
				clientIP,
				statusCode,
				latency,
				c.Errors.String(),
			)
		}
	}
}
