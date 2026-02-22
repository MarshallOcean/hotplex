# HotPlex 示例方案

🌐 [English Version](README.md)

本目录包含了如何使用 HotPlex SDK 和代理服务器的各类示例。

## 📁 示例结构

### 1. [Claude 基础 (Go)](./go_claude_basic)
演示如何使用 `HotPlexClient` 与 Claude Code CLI 进行基础交互的简单 Go 程序。

### 2. [Claude 生命周期 (Go)](./go_claude_lifecycle)
一个完整的 Go 示例，展示了 Claude 会话的全生命周期：
- **冷启动 (Cold Start)**: 初始化一个新的持久化进程。
- **热复用 (Hot-Multiplexing)**: 复用现有进程，实现亚秒级响应。
- **进程恢复**: 如何在进程异常退出后，利用标记文件恢复会话。
- **手动终止**: 显式停止会话。

### 3. [OpenCode 基础 (Go)](./go_opencode_basic)
演示如何将 HotPlex 与 **OpenCode** CLI 智能体配合使用：
- **提供商切换**: 无缝更换底层的 AI 智能体。
- **Plan/Build 模式**: 配置 OpenCode 特有的运行模式。
- **模型配置**: 覆盖默认的模型设置。

### 4. [OpenCode 生命周期 (Go)](./go_opencode_lifecycle)
一个全面的 Go 示例，展示了 OpenCode 会话的全生命周期：
- **冷启动**: 使用 GLM-5 模型初始化新的持久化进程。
- **多轮交互**: 在同一个会话中进行持续对话。
- **会话持久化**: HotPlex 如何维护特定提供商的会话状态。
- **温启动恢复**: 使用 SessionID 恢复之前的会话。

### 5. [OpenCode HTTP 客户端 (Python)](./python_opencode_http)
演示 OpenCode 兼容层的 REST + SSE 交互模式的 Python 客户端：
- **SSE 监听**: 使用 `requests` 实现实时事件流处理。
- **会话管理**: RESTful 风格的会话创建与消息推送。
- **事件解析**: 将 OpenCode 的 "Parts" 架构解析为更直观的控制台输出。

### 6. [Claude WebSocket 客户端 (Node.js)](./node_claude_websocket)

| 文件                   | 描述                                                                  |
| :--------------------- | :-------------------------------------------------------------------- |
| `client.js`            | **快速上手** - 仅约 50 行代码，30 秒即可跑通                          |
| `enterprise_client.js` | **企业级** - 具备重连、改进错误处理、指标监控和优雅停机的生产级客户端 |

**企业级特性：**
- 具备指数退避逻辑的自动重连
- 全面的错误处理与恢复机制
- 可配置层级的结构化日志
- 连接健康监测 (心跳)
- 请求超时管理
- 优雅停机支持 (SIGINT/SIGTERM)
- 指标收集 (延迟、成功率、重连次数)
- 针对流式事件的进度回调

---

## 🚀 如何运行

### 先决条件：安装 Claude Code CLI
请确保已安装 `claude` CLI 并完成认证。

#### 推荐安装方式 (原生):
```bash
# macOS / Linux / WSL
curl -fsSL https://claude.ai/install.sh | bash

# Windows (PowerShell)
irm https://claude.ai/install.ps1 | iex
```

#### 其他方式:
```bash
brew install claude-code
# 或
npm install -g @anthropic-ai/claude-code
```

执行认证：
```bash
claude auth
```

### 运行 Go 示例
```bash
# Claude 基础演示
go run _examples/go_claude_basic/main.go

# Claude 生命周期演示
go run _examples/go_claude_lifecycle/main.go

# OpenCode 基础演示
go run _examples/go_opencode_basic/main.go

# OpenCode 生命周期演示
go run _examples/go_opencode_lifecycle/main.go
```

### 运行 Python 示例
```bash
cd _examples/python_opencode_http
pip install requests
python client.py
```

### 运行 WebSocket 示例

1. 启动 HotPlex 代理服务器：
   ```bash
   go run cmd/hotplexd/main.go
   ```

2. 运行 Node.js 客户端（在另一个终端）：
   ```bash
   cd _examples/node_claude_websocket
   npm install

   # 快速上手
   node client.js

   # 企业级演示
   node enterprise_client.js
   ```

### 运行 OpenCode HTTP API (cURL)

验证 OpenCode 兼容层：
```bash
# 1. 创建会话
curl -X POST http://localhost:8080/session

# 2. 建立事件流 (在独立终端中运行)
curl -N http://localhost:8080/global/event

# 3. 发送 Prompt
curl -X POST http://localhost:8080/session/<id>/message \
     -H "Content-Type: application/json" \
     -d '{"prompt": "写一个 Python 脚本来列出文件"}'
```

### 将企业级客户端作为模块使用
```javascript
const { HotPlexClient } = require('./enterprise_client');

const client = new HotPlexClient({
  url: 'ws://localhost:8080/ws/v1/agent',
  sessionId: 'my-session',
  logLevel: 'info',
  reconnect: { enabled: true, maxAttempts: 5 }
});

await client.connect();

const result = await client.execute('列出当前目录的文件', {
  systemPrompt: '你是一个得力的助手。',
  onProgress: (event) => {
    if (event.type === 'answer') process.stdout.write(event.data);
  }
});

console.log(result);
await client.disconnect();
```

## 📡 协议说明

### 请求-响应关联 (Correlation)
所有 WebSocket 请求都支持可选的 `request_id` 字段。服务器会在响应中返回相同的 ID，以便在同一个连接上处理并发请求时进行关联。

```javascript
// 带有 request_id 的请求
{ "request_id": 123, "type": "execute", "prompt": "..." }

// 响应会包含相同的 request_id
{ "request_id": 123, "event": "answer", "data": "..." }
```

## ⚙️ 配置提示
- **`IDLE_TIMEOUT`**: 运行 `hotplexd` 时设置此环境变量可更改闲置进程的存活时间（例如：`IDLE_TIMEOUT=5m`）。
- **`PORT`**: 更改默认的 `8080` 端口。
