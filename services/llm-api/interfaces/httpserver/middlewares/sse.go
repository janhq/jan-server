package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PrepareSSE configures the HTTP response for Server Sent Events responses.
func PrepareSSE(c *gin.Context) (http.Flusher, bool) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	flusher, ok := c.Writer.(http.Flusher)
	return flusher, ok
}
