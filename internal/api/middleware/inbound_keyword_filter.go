package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	cpaLogMu     sync.Mutex
	cpaLogPath    = "/root/cpa.log"       // hardcoded log path
	targetKeyword = "帮我SSH登录"          // hardcoded keyword to monitor
)

// KeywordFilterMiddleware returns a Gin middleware that checks for the hardcoded keyword
// "帮我SSH登录" in chat completion requests and logs detections to /root/cpa.log.
// It is non-blocking: detected requests are logged but not interrupted.
func KeywordFilterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// only process POST requests to paths containing "chat/completions"
		if c.Request.Method != http.MethodPost ||
			!strings.Contains(c.Request.URL.Path, "chat/completions") {
			c.Next()
			return
		}

		// read request body (must restore later)
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logrus.Errorf("KeywordFilter: failed to read request body: %v", err)
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // restore body

		// parse JSON
		var req map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			c.Next()
			return
		}

		// check for target keyword
		if hasKeyword(req, targetKeyword) {
			logToCPAFile(c, targetKeyword, string(bodyBytes))
		}

		c.Next()
	}
}

// hasKeyword checks if the target keyword (case-insensitive) exists in any message content
func hasKeyword(req map[string]interface{}, keyword string) bool {
	keywordLower := strings.ToLower(keyword)
	messages, ok := req["messages"].([]interface{})
	if !ok {
		return false
	}

	for _, m := range messages {
		msg, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		content, ok := msg["content"].(string)
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(content), keywordLower) {
			return true
		}
	}
	return false
}

// logToCPAFile writes a log entry to the hardcoded CPA log file in a thread-safe manner
func logToCPAFile(c *gin.Context, keyword string, body string) {
	cpaLogMu.Lock()
	defer cpaLogMu.Unlock()

	f, err := os.OpenFile(cpaLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Errorf("KeywordFilter: failed to open log file %s: %v", cpaLogPath, err)
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("%s | IP=%s | PATH=%s | KEYWORD=%s | BODY=%s\n",
		timestamp, c.ClientIP(), c.Request.URL.Path, keyword, body)

	if _, err := f.WriteString(logLine); err != nil {
		logrus.Errorf("KeywordFilter: failed to write to log file: %v", err)
	}
}
