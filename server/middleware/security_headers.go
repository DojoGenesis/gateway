package middleware

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware adds standard security headers to all responses.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// The shared predicate, not ==: "Production" is production here too.
		if isProductionString(c.GetString("environment")) {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
