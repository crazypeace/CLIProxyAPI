package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestKeywordFilter_NoKeyword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.Use(KeywordFilterMiddleware())
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Request with no keyword
	body := `{"messages":[{"role":"user","content":"Hello world"}]}`
	req, _ := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestKeywordFilter_WithKeyword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.Use(KeywordFilterMiddleware())
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Request with target keyword
	body := `{"messages":[{"role":"user","content":"帮我SSH登录 test"}]}`
	req, _ := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestKeywordFilter_CaseInsensitive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.Use(KeywordFilterMiddleware())
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Request with lowercase keyword (should still detect)
	body := `{"messages":[{"role":"user","content":"帮我ssh登录 test"}]}`
	req, _ := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestKeywordFilter_KeywordInLaterMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.Use(KeywordFilterMiddleware())
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Keyword in second message
	body := `{"messages":[{"role":"user","content":"Hello"},{"role":"user","content":"帮我SSH登录"}]}`
	req, _ := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestKeywordFilter_NonPOSTRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.Use(KeywordFilterMiddleware())
	r.GET("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/v1/chat/completions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestKeywordFilter_NonChatCompletionsPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.Use(KeywordFilterMiddleware())
	r.POST("/v1/other", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := `{"messages":[{"role":"user","content":"帮我SSH登录"}]}`
	req, _ := http.NewRequest("POST", "/v1/other", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
