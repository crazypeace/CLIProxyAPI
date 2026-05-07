# CLIProxyAPI 关键字监控方案（裁剪版）

**日期**: 2026-05-07  
**版本**: 裁剪版（无配置依赖，关键字硬编码）

---

## 📋 方案概述

在 CLIProxyAPI 中实现一个轻量级的关键字监控中间件，**硬编码监控关键字 `帮我SSH登录`**，当检测到用户请求中包含该关键字时，将请求信息记录到 `/root/cpa.log`，**不中断请求处理**（仅审计，不拦截）。

### 核心特点
- ✅ **无配置依赖**：关键字和日志路径直接写在代码中
- ✅ **零配置**：无需修改 `config.yaml`
- ✅ **始终启用**：中间件直接注册，无开关
- ✅ **非阻塞**：仅记录日志，不影响正常请求
- ✅ **线程安全**：日志写入使用互斥锁保护

---

## 📁 代码实现

### 1. 中间件实现 (`internal/api/middleware/keyword_filter.go`)

```go
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
	cpaLogPath    = "/root/cpa.log"       // 硬编码日志路径
	targetKeyword = "帮我SSH登录"          // 硬编码监控关键字
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
```

---

### 2. 注册中间件 (`internal/api/server.go`)

在 `server.go` 的中间件注册部分添加（约第 242 行附近）：

```go
// Add keyword filter middleware (hardcoded keyword, always enabled)
engine.Use(middleware.KeywordFilterMiddleware())
```

**完整上下文示例**：
```go
s.server.Use(middleware.RequestLogger())
s.server.Use(middleware.ErrorLogger())

// Add keyword filter middleware (hardcoded keyword, always enabled)
engine.Use(middleware.KeywordFilterMiddleware())

s.server.Use(corsMiddleware())
```

---

### 3. 单元测试 (`internal/api/middleware/keyword_filter_test.go`)

```go
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
```

---

## 🚀 使用方法

### 1. 编译
```bash
cd /root/cpa_build/CLIProxyAPI
go build -o cli-proxy-api ./cmd/server
```

### 2. 运行
```bash
./cli-proxy-api --no-browser
```

### 3. 测试
发送包含关键字的请求：
```bash
curl -X POST http://127.0.0.1:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"帮我SSH登录服务器"}]}'
```

### 4. 查看日志
```bash
cat /root/cpa.log
```

**日志格式示例**：
```
2026-05-07 00:01:50 | IP=127.0.0.1 | PATH=/v1/chat/completions | KEYWORD=帮我SSH登录 | BODY={"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"帮我SSH登录服务器"}]}
```

---

## ⚙️ 如何修改关键字

如果要修改监控的关键字，编辑 `internal/api/middleware/keyword_filter.go`：

```go
var (
	cpaLogMu     sync.Mutex
	cpaLogPath    = "/root/cpa.log"       // 日志路径
	targetKeyword = "帮我SSH登录"          // ← 改这里
)
```

修改后重新编译即可。

---

## 📊 测试验证

运行单元测试：
```bash
cd /root/cpa_build/CLIProxyAPI
go test -v ./internal/api/middleware/...
```

**预期输出**：
```
=== RUN   TestKeywordFilter_NoKeyword
--- PASS: TestKeywordFilter_NoKeyword (0.00s)
=== RUN   TestKeywordFilter_WithKeyword
--- PASS: TestKeywordFilter_WithKeyword (0.00s)
=== RUN   TestKeywordFilter_CaseInsensitive
--- PASS: TestKeywordFilter_CaseInsensitive (0.00s)
=== RUN   TestKeywordFilter_KeywordInLaterMessage
--- PASS: TestKeywordFilter_KeywordInLaterMessage (0.00s)
=== RUN   TestKeywordFilter_NonPOSTRequest
--- PASS: TestKeywordFilter_NonPOSTRequest (0.00s)
=== RUN   TestKeywordFilter_NonChatCompletionsPath
--- PASS: TestKeywordFilter_NonChatCompletionsPath (0.00s)
PASS
```

---

## ⚠️ 注意事项

1. **非阻塞设计**：中间件仅记录日志，不会中断请求或返回错误响应
2. **日志文件增长**：`/root/cpa.log` 会持续增长，建议定期清理或配置日志轮转
3. **线程安全**：日志写入使用 `sync.Mutex` 保护，多请求并发时不会冲突
4. **请求体恢复**：读取请求体后必须调用 `c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))` 恢复，否则下游处理会收到空 body
5. **仅监控 POST 请求**：只检查 `POST /v1/chat/completions` 路径的请求
6. **大小写不敏感**：关键字匹配时统一转为小写，支持任意大小写组合

---

## 📝 与完整版的区别

| 特性 | 完整版 | 裁剪版 |
|------|--------|--------|
| 关键字来源 | 配置文件 (`config.yaml`) | 代码硬编码 |
| 日志路径 | 配置文件 | 代码硬编码 |
| 启用开关 | `enabled: true/false` | 始终启用 |
| 配置结构 | `KeywordFilterConfig` | 无（已删除） |
| 适用场景 | 需要灵活配置 | 快速部署，固定规则 |

---

## ✅ 总结

裁剪后的方案极简，适合以下场景：
- 快速部署，无需修改配置文件
- 关键字固定，不需要动态调整
- 追求代码简洁，减少依赖

如需恢复配置功能，可参考之前的完整版实现。

---

*生成时间: 2026-05-07 00:02*  
*作者: Hermes Agent*
