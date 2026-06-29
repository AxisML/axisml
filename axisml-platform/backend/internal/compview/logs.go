package compview

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
)

// LogOptions parses the contract pod-log query knobs.
func LogOptions(c *gin.Context) computeservice.LogOptions {
	opt := computeservice.LogOptions{
		Container: c.Query("container"),
		Follow:    c.Query("follow") == "true",
		Previous:  c.Query("previous") == "true",
	}
	if v := c.Query("tailLines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opt.TailLines = &n
		}
	}
	return opt
}

// StreamLogs pipes an upstream log response to the client, preserving the
// upstream content type (text/plain one-shot, or text/event-stream for SSE
// follow) and flushing as data arrives. It closes resp.Body.
func StreamLogs(c *gin.Context, resp *http.Response) {
	defer func() { _ = resp.Body.Close() }()
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "text/plain; charset=utf-8"
	}
	c.Writer.Header().Set("Content-Type", ct)
	c.Status(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := c.Writer.Write(buf[:n]); werr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			return
		}
	}
}
