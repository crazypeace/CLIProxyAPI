# 使用 for-agentrouter-d4f 版 CPA 调用 agentrouter 的 deepseek-v4-flash 模型

> 本指南面向 **已经在本地运行 CPA 官方版本、且已经配置好 agentrouter** 的用户。
> 用户在调用 deepseek-v4-flash 模型时, 多轮对话从第二轮开始报
> `HTTP 400: The content[].thinking in the thinking mode must be passed back to the API`。
> `for-agentrouter-d4f` 增加了 request-format: anthropic 选项, 把 OpenAI 客户端请求翻译成 Anthropic
> Messages 协议后再发给 agentrouter。

本指南 **不会修改你原有的 CPA 官方版**, 也不会动它的目录、日志、配置或 auth 文件。
将 for-agentrouter-d4f 部署为一个**完全独立**的新实例。

---

## 1. 准备: 拿到修复版的二进制

从 GitHub Releases 页面下载对应平台的二进制:

- Release 页: <https://github.com/crazypeace/CLIProxyAPI/releases/tag/for-agentrouter-d4f>

| 平台 | 下载链接 |
|---|---|
| linux amd64 (x86_64) | <https://github.com/crazypeace/CLIProxyAPI/releases/download/for-agentrouter-d4f/CLIProxyAPI-linux-amd64> |
| linux arm64 (aarch64) | <https://github.com/crazypeace/CLIProxyAPI/releases/download/for-agentrouter-d4f/CLIProxyAPI-linux-arm64> |

---

## 2. 设计原则: 与原版 CPA 完全隔离

为避免覆盖原版 CPA 的端口、auth 文件、日志, 修复版推荐使用:

| 项目 | 原版 CPA | 修复版 CPA |
|---|---|---|
| 二进制 | cli-proxy-api (任意路径) | cli-proxy-api-patched (建议改名) |
| 配置文件 | config.yaml | config-patched.yaml |
| auth 文件目录 | ~/.cli-proxy-api/ | ~/.cli-proxy-api-patched/ |
| HTTP 监听端口 | 8317 (默认) | 9417 (建议) |
| pprof 端口 | 8316 (默认) | 9416 (建议) |
| 日志目录 | ~/.cli-proxy-api/logs/ | ~/.cli-proxy-api-patched/logs/ |
| 运行身份 | 任意用户, 任意 HOME | 任意用户, 但通过 --config 显式指定配置 |

只要做到上面这些, 修复版和原版可以**同时运行**。

---

## 3. 一次性部署步骤

### 3.1 准备目录与二进制

```bash
# 创建独立的工作目录
mkdir -p ~/cpa-patched
cd ~/cpa-patched

# 把下载的二进制放进来, 并改名避免和原版混淆
mv ~/Downloads/CLIProxyAPI-linux-amd64 ./cli-proxy-api-patched
chmod +x ./cli-proxy-api-patched

# 创建独立的 auth 目录
mkdir -p ~/.cli-proxy-api-patched
```

> linux-arm64 用户把上面文件名换成 `CLIProxyAPI-linux-arm64` 即可。

### 3.2 编写配置文件

把下面这份配置保存为 `~/cpa-patched/config-patched.yaml`。

**配置项分成两类**:

- **A. 从原版 config.yaml 复制** — 这些值是用户在原版 CPA 里已经调过的, 直接照抄过来, 不要随便改:
  - commercial-mode — 按原版的设定
  - debug — 按原版的设定
  - logging-to-file / error-logs-max-files — 按原版的设定
  - routing.strategy — 按原版的设定
  - request-retry / max-retry-credentials / max-retry-interval — 按原版的设定
  - quota-exceeded.* — 按原版的设定
  - remote-management.* — 按原版的设定 (如果有改动)
  - tls.* — 按原版的设定
  - api-keys — 原版用什么, 这里就用什么
  - **`openai-compatibility` 整段** (含 name / base-url / api-key-entries / headers / models) — 必须原样从原版 config.yaml 里复制过来
- **B. 按本指南设置** — 这些值是为了让修复版独立运行, 跟原版 CPA 不冲突:
  - auth-dir → ${HOME}/.cli-proxy-api-patched (独立目录)
  - host → 127.0.0.1 (本地监听)
  - port → 9417 (与原版 8317 错开)
  - pprof.addr → 127.0.0.1:9416 (与原版 8316 错开)
  - pprof.enable → 按需, 默认 false
  - **`request-format: anthropic`** ← 本次修复的核心开关, 必须加上
  - passthrough-headers / ws-auth — 按需

下面这份模板是**最小骨架**, 复制原版 config.yaml 后, 把 B 类按这里写、A 类按原版照抄即可:

```yaml
api-keys:
  - "<原版 api-keys 数组中的字符串>"

auth-dir: ${HOME}/.cli-proxy-api-patched
commercial-mode: <原版值>
debug: <原版值>
host: 127.0.0.1
port: 9417

pprof:
  addr: 127.0.0.1:9416
  enable: false

logging-to-file: <原版值>
error-logs-max-files: <原版值>

# ↓↓↓ openai-compatibility 整段从原版 config.yaml 复制过来 ↓↓↓
openai-compatibility:
  - name: agentrouter
    base-url: https://agentrouter.org/v1   # 原版怎么写就怎么写
    api-key-entries:                       # 原版怎么写就怎么写
      - api-key: "<原版的 agentrouter api key>"
    headers:                               # 原版怎么写就怎么写, 不要改
      User-Agent: Kilo-Code/7.3.50 ai-sdk/provider-utils/4.0.27 runtime/bun/1.3.14
    models:                                # 原版怎么写就怎么写
      - name: deepseek-v4-flash
        alias: ""
    disabled: false                        # 原版这里可能是 true (出错后用户会暂时禁用), 修复版这里要 false
    # ↓↓↓ 本次修复的开关: 不加这一行, 行为和原版一样, 依然 400
    request-format: anthropic

routing:
  strategy: <原版值>
request-retry: <原版值>
max-retry-credentials: <原版值>
max-retry-interval: <原版值>

remote-management:
  allow-remote: <原版值>
  disable-control-panel: <原版值>
  secret-key: "<原版值>"

quota-exceeded:
  switch-preview-model: <原版值>
  switch-project: <原版值>

passthrough-headers: <原版值>
ws-auth: <原版值>
tls:
  enable: <原版值>
```

### 3.3 启动修复版

```bash
cd ~/cpa-patched
./cli-proxy-api-patched --config ./config-patched.yaml
```

启动成功会看到类似输出:

```
CLIProxyAPI Version: ..., Commit: ..., BuiltAt: ...
API server started successfully on: 127.0.0.1:9417
server clients and configuration updated: 1 clients (... + 1 OpenAI-compat)
```

如果报端口冲突, 说明 port: 9417 也被占了, 改成一个空闲端口即可。

### 3.4 验证修复版能正常代理

**检查模型列表**:

```bash
CPA_KEY="<原版 api-keys 数组中的字符串>"
curl -sS http://127.0.0.1:9417/v1/models \
  -H "Authorization: Bearer ${CPA_KEY}" | head -c 500
```

应该能看到 deepseek-v4-flash 在返回的模型列表里。

**对话测试** (单轮, 用来确认 key + UA + 翻译链路全打通):

```bash
curl -sS http://127.0.0.1:9417/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${CPA_KEY}" \
  -d '{
    "model": "deepseek-v4-flash",
    "messages": [{"role": "user", "content": "reply with exactly: ok"}]
  }' | head -c 500
```

期望返回 ok, 且**不要看到 HTTP 400**。

### 3.5 如何使用修复版

base_url:

```
http://127.0.0.1:9417/v1    (修复版)
```

api_key 用 config-patched.yaml 里 api-keys 那项的值。

model 名不变 (deepseek-v4-flash)


> ⚠️ 不要 在已经报错 400 的 session 中切换到 新CPA继续对话.
> new 一个新 session 再使用 新CPA.
