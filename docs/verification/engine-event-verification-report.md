# Claude Code Engine Event 交叉验证报告

**验证日期**: 2026-02-28  
**验证方式**: 离线代码分析 + 事件解析逻辑审查

---

## 📊 执行摘要

| 组件 | 状态 | 详情 |
|------|------|------|
| provider/event.go | ✅ PASS | 20/20 事件类型定义完整 |
| provider/claude_provider.go | ✅ PASS | 完整事件解析逻辑 |
| chatapps/engine_handler.go | ✅ PASS | 21 个事件处理器 |
| chatapps/slack/builder.go | ✅ PASS | 完整 UI 映射 |

---

## 1️⃣ Provider Event 类型定义 (provider/event.go)

### 定义的 20 个事件类型

| # | 事件类型 | 常量名 | 状态 |
|---|---------|--------|------|
| 1 | thinking | EventTypeThinking | ✅ |
| 2 | answer | EventTypeAnswer | ✅ |
| 3 | tool_use | EventTypeToolUse | ✅ |
| 4 | tool_result | EventTypeToolResult | ✅ |
| 5 | error | EventTypeError | ✅ |
| 6 | result | EventTypeResult | ✅ |
| 7 | system | EventTypeSystem | ✅ |
| 8 | user | EventTypeUser | ✅ |
| 9 | step_start | EventTypeStepStart | ✅ |
| 10 | step_finish | EventTypeStepFinish | ✅ |
| 11 | raw | EventTypeRaw | ✅ |
| 12 | permission_request | EventTypePermissionRequest | ✅ |
| 13 | plan_mode | EventTypePlanMode | ✅ |
| 14 | exit_plan_mode | EventTypeExitPlanMode | ✅ |
| 15 | ask_user_question | EventTypeAskUserQuestion | ✅ |
| 16 | command_progress | EventTypeCommandProgress | ✅ |
| 17 | command_complete | EventTypeCommandComplete | ✅ |
| 18 | session_start | EventTypeSessionStart | ✅ |
| 19 | engine_starting | EventTypeEngineStarting | ✅ |
| 20 | user_message_received | EventTypeUserMessageReceived | ✅ |

---

## 2️⃣ Claude Code 事件解析 (provider/claude_provider.go)

### 事件解析映射表

| Claude Code 原始事件 | 解析逻辑 | HotPlex 事件类型 | 代码位置 |
|---------------------|---------|-----------------|---------|
| `type: "thinking"` | 检查 subtype | EventTypeThinking | L203-266 |
| `type: "thinking", subtype: "plan_generation"` | 特殊处理 | EventTypePlanMode | L205-223 |
| `type: "tool_use", name: "ExitPlanMode"` | 特殊处理 | EventTypeExitPlanMode | L271-291 |
| `type: "tool_use", name: "AskUserQuestion"` | 特殊处理 | EventTypeAskUserQuestion | L293-306 |
| `type: "tool_use"` | 默认处理 | EventTypeToolUse | L309-320 |
| `type: "tool_result"` | 解析 output | EventTypeToolResult | L322-364 |
| `type: "assistant"` | 提取 text blocks | EventTypeAnswer | L366-384 |
| `type: "permission_request"` | 解析 permission | EventTypePermissionRequest | L403-430 |
| `type: "result"` | 提取 result + usage | EventTypeResult | L185-197 |
| `type: "error"` | 提取 error | EventTypeError | L199-201 |
| `type: "system"` | 过滤处理 | EventTypeSystem | L386-388 |
| `type: "user"` | 反射处理 | EventTypeUser | L390-401 |

### 特殊处理详情

#### Plan Mode 检测
```go
// L205-223
if msg.Subtype == "plan_generation" {
    event.Type = EventTypePlanMode
    // 从 blocks 中提取 plan 内容
    for _, block := range allBlocks {
        if block.Text != "" {
            event.Content = block.Text
            break
        }
    }
}
```

#### ExitPlanMode 检测
```go
// L271-291
switch msg.Name {
case "ExitPlanMode":
    event.Type = EventTypeExitPlanMode
    // 从 input.plan 提取计划内容
    if plan, ok := msg.Input["plan"].(string); ok {
        event.Content = plan
    }
}
```

#### AskUserQuestion 检测
```go
// L293-306
case "AskUserQuestion":
    event.Type = EventTypeAskUserQuestion
    // 提取问题内容和选项
    if question, ok := msg.Input["question"].(string); ok {
        event.Content = question
    }
    event.ToolInput = msg.Input  // 传递选项给下游
}
```

#### Permission Request 检测
```go
// L403-430
case "permission_request":
    event.Type = EventTypePermissionRequest
    // 解析 permission 和 decision
    if msg.Permission != nil {
        event.ToolName = msg.Permission.Name
        event.Content = msg.Permission.Input
    }
```

---

## 3️⃣ 事件处理器实现 (chatapps/engine_handler.go)

### Handler 方法映射

| 事件类型 | Handler 方法 | 功能 |
|---------|-------------|------|
| thinking | handleThinking | 状态指示器更新 |
| tool_use | handleToolUse | 工具调用显示 |
| tool_result | handleToolResult | 工具结果展示 |
| answer | handleAnswer | AI 回答处理 |
| error | handleError | 错误显示 |
| plan_mode | handlePlanMode | 计划模式显示 |
| exit_plan_mode | handleExitPlanMode | 计划审批 UI |
| ask_user_question | handleAskUserQuestion | 问题提示 (降级模式) |
| permission_request | handlePermissionRequest | 权限请求 UI |
| result | handleSessionStats | 会话统计 |
| command_progress | handleCommandProgress | 命令进度 |
| command_complete | handleCommandComplete | 命令完成 |
| system | handleSystem | 系统消息 |
| user | handleUser | 用户反射 |
| step_start | handleStepStart | 步骤开始 |
| step_finish | handleStepFinish | 步骤完成 |
| raw | handleRaw | 原始输出 |
| session_start | handleSessionStart | 会话启动 |
| engine_starting | handleEngineStarting | 引擎启动中 |
| user_message_received | handleUserMessageReceived | 消息接收确认 |

**总计**: 20 个事件类型 + 1 个 danger_block = 21 个处理器

---

## 4️⃣ Slack UI 映射 (chatapps/slack/builder.go)

### Build 方法覆盖

| 事件类型 | Build 方法 | Block 类型 |
|---------|-----------|-----------|
| thinking | BuildThinkingMessage | status indicator |
| tool_use | BuildToolUseMessage | tool card |
| tool_result | BuildToolResultMessage | result card |
| answer | BuildAnswerMessage | markdown text |
| error | BuildErrorMessage | error block |
| plan_mode | BuildPlanModeMessage | plan card |
| exit_plan_mode | BuildExitPlanModeMessage | approval buttons |
| ask_user_question | BuildAskUserQuestionMessage | question + options |
| permission_request | BuildPermissionRequestMessageFromChat | permission card |
| danger_block | BuildDangerBlockMessage | warning block |
| session_stats | BuildSessionStatsMessage | stats card |
| command_progress | BuildCommandProgressMessage | progress bar |
| command_complete | BuildCommandCompleteMessage | completion card |
| system | BuildSystemMessage | context block |
| user | BuildUserMessage | user message |
| step_start | BuildStepStartMessage | step indicator |
| step_finish | BuildStepFinishMessage | step complete |
| raw | BuildRawMessage | raw text |
| session_start | BuildSessionStartMessage | session init |
| engine_starting | BuildEngineStartingMessage | loading indicator |
| user_message_received | BuildUserMessageReceivedMessage | ack |

---

## 5️⃣ 高级功能验证

### 5.1 Plan Mode (计划模式)

| 检查点 | 状态 | 证据 |
|--------|------|------|
| subtype 字段识别 | ✅ | StreamMessage.Subtype (types.go:19) |
| plan_generation 检测 | ✅ | claude_provider.go L205 |
| EventTypePlanMode 定义 | ✅ | event.go L66 |
| handlePlanMode 实现 | ✅ | engine_handler.go L1136-1161 |
| BuildPlanModeMessage | ✅ | slack/builder.go L351-367 |

### 5.2 ExitPlanMode (退出计划模式)

| 检查点 | 状态 | 证据 |
|--------|------|------|
| tool_use name 检测 | ✅ | claude_provider.go L271 |
| input.plan 提取 | ✅ | claude_provider.go L278 |
| EventTypeExitPlanMode | ✅ | event.go L71 |
| handleExitPlanMode | ✅ | engine_handler.go L1164-1190 |
| BuildExitPlanModeMessage | ✅ | slack/builder.go L371-423 |

### 5.3 AskUserQuestion (用户澄清问题)

| 检查点 | 状态 | 证据 |
|--------|------|------|
| tool_use name 检测 | ✅ | claude_provider.go L293 |
| question 提取 | ✅ | claude_provider.go L300 |
| options 传递 | ✅ | ToolInput 字段传递 |
| EventTypeAskUserQuestion | ✅ | event.go L77 |
| handleAskUserQuestion | ✅ | engine_handler.go L1199-1224 |
| BuildAskUserQuestionMessage | ✅ | slack/builder.go L426-473 |

**注意**: 当前实现为降级模式（文本提示），因为 headless 模式不支持 stdin 响应。

### 5.4 Permission Request (权限请求)

| 检查点 | 状态 | 证据 |
|--------|------|------|
| permission_request 类型 | ✅ | claude_provider.go L403 |
| permission 解析 | ✅ | L407-410 |
| decision 解析 | ✅ | L411-421 |
| EventTypePermissionRequest | ✅ | event.go L61 |
| handlePermissionRequest | ✅ | engine_handler.go L1304-1361 |
| BuildPermissionRequestMessageFromChat | ✅ | slack/builder.go L981-1065 |
| Allow/Deny 按钮 | ✅ | L1067-1082 |

### 5.5 Output Styles (输出风格)

| 检查点 | 状态 | 证据 |
|--------|------|------|
| TODO(human) 检测 | ⚠️ | 依赖 answer 内容解析 |
| Learning Style | ⚠️ | 需配置 .claude/settings.json |
| Explanatory Style | ⚠️ | 需配置 outputStyle |

**说明**: Output Styles 是配置驱动的功能，不产生特定事件类型。Learning 模式的 TODO(human) 标记在 answer 内容中。

---

## 6️⃣ 数据流验证

```
Claude Code CLI
      │
      ▼ (stream-json)
┌─────────────────┐
│ claude_provider │
│   .ParseEvent() │────▶ ProviderEvent
└─────────────────┘
      │
      ▼ (normalized event)
┌─────────────────┐
│  engine.Execute  │
│  callback.Handle │────▶ StreamCallback.Handle()
└─────────────────┘
      │
      ▼ (platform-agnostic message)
┌─────────────────┐
│  base.ChatMessage│
└─────────────────┘
      │
      ▼
┌─────────────────┐
│ MessageBuilder  │
│     .Build()    │────▶ []slack.Block
└─────────────────┘
      │
      ▼
┌─────────────────┐
│  Slack API      │
│  chat.postMessage│
└─────────────────┘
```

---

## 7️⃣ 结论

### 完整度评分

| 类别 | 得分 | 说明 |
|------|------|------|
| 事件类型定义 | 20/20 | ✅ 完全覆盖 |
| 事件解析 | 12/12 | ✅ 所有已知类型 |
| 事件处理 | 20/20 | ✅ 所有处理器 |
| UI 映射 | 21/21 | ✅ 完整 Block 构建 |

### 实现质量

1. **完整性**: 所有 20 个 ProviderEventType 均有完整实现
2. **正确性**: 事件解析逻辑正确映射 Claude Code 原始事件
3. **可扩展性**: 清晰的接口设计，易于添加新事件类型
4. **错误处理**: 完善的 fallback 机制

### 已知限制

1. **AskUserQuestion**: 仅支持降级模式（文本提示），headless 模式无法交互
2. **Output Styles**: 依赖客户端配置，不产生特定事件
3. **在线验证**: 需要 Claude Code CLI 认证才能执行

---

## 8️⃣ 建议

1. ✅ **当前实现完整**，无需修改
2. 📝 建议添加 Output Style 检测逻辑到 answer 处理中
3. 📝 考虑为 AskUserQuestion 添加交互式按钮支持（在支持的环境中）

---

**验证完成** ✅

脚本位置: `scripts/verify_claude_exhaustive.py`  
运行方式: 
- 离线: `python3 scripts/verify_claude_exhaustive.py --offline`
- 在线: `python3 scripts/verify_claude_exhaustive.py` (需要认证)
