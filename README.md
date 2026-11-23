# go-agent

一个用 Go 实现的命令行 AI Agent：接入大模型，通过工具注册表完成文件读写、受限命令执行、长期记忆、Skill 加载，并通过 MCP 协议接入 Playwright 浏览器自动化。提供 HTTP API 与 Bubble Tea 终端聊天两种交互方式。

## 功能特性

- **工具注册表**：统一的 `Tool` 接口，本地工具与外部 MCP 工具以同一套接口注册、调用。
- **LLM 对话闭环**：模型返回工具调用后，由 Agent 循环执行工具、把结果回灌上下文，直到给出最终回复。
- **长期记忆**：按会话持久化对话历史；关键偏好/约定以 Notes 分层长期保留，历史消息做条数与长度截断，避免无限膨胀。
- **Skill 机制**：从 `skills/` 目录加载 Markdown 规范，拼接进系统提示词。
- **MCP 接入**：通过官方 MCP Go SDK 以 stdio 子进程连接 Playwright MCP，发现并注册浏览器工具。
- **两种入口**：HTTP 服务（`/health`、`/chat`，Bearer 鉴权）与终端 TUI（Bubble Tea）。
- **安全边界**：文件路径穿越 / 符号链接 / 绝对路径拦截，Shell 命令白名单，MCP 返回内容仅作文本处理。

## 目录结构

```
.
├── main.go                     # 程序入口：组装 Agent，分发 HTTP / TUI 子命令
├── config/                     # 环境变量与 .env 配置加载
├── internal/
│   ├── agent/                  # 对话循环、工具定义、系统提示词
│   ├── llm/                    # 模型接口、DeepSeek 兼容客户端、测试用 Mock
│   ├── tools/                  # 工具接口、注册表、file / shell 工具
│   ├── mcp/                    # MCP 客户端、适配器、工具发现、跨平台命令构造
│   ├── memory/                # 会话记忆存储、历史压缩、关键信息 Notes
│   ├── skills/                # Skill Markdown 加载与提示词拼接
│   ├── server/                 # HTTP 路由、鉴权中间件
│   └── tui/                    # Bubble Tea 终端聊天界面
├── skills/                     # 内置 Skill 规范（网页开发）
├── workspace/                  # Agent 生成的网页等产物（运行时，不入库）
└── memory/                     # 会话记忆数据（运行时，不入库）
```

## 环境要求

- Go 1.25+
- Node.js 20+（用于启动 Playwright MCP 子进程）
- 一个兼容 OpenAI `/chat/completions` 接口的模型服务及 API Key

## 快速开始

### 1. 配置环境变量

复制示例配置并填写 Key：

```powershell
Copy-Item .env.example .env
```

编辑 `.env`：

```env
DEEPSEEK_API_KEY=你的API_Key
DEEPSEEK_MODEL=deepseek-flash
DEEPSEEK_BASE_URL=https://tokenrhythm.studio/v1
```

### 2. 启动

**HTTP 服务**（默认监听 `127.0.0.1:8095`）：

```powershell
go run .
```

**终端聊天界面**：

```powershell
go run . tui
```

### 3. 调用 HTTP 接口

健康检查：

```powershell
Invoke-RestMethod http://127.0.0.1:8095/health
```

发送对话：

```powershell
$body = @{ message = "帮我做一个个人主页"; session_id = "demo" } | ConvertTo-Json
Invoke-RestMethod -Uri http://127.0.0.1:8095/chat -Method Post -Body $body -ContentType "application/json; charset=utf-8"
```

## 配置项

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `DEEPSEEK_API_KEY` | （必填） | 模型服务凭证 |
| `DEEPSEEK_MODEL` | `deepseek-flash` | 模型名称 |
| `DEEPSEEK_BASE_URL` | `https://tokenrhythm.studio/v1` | 兼容 OpenAI 接口的服务地址 |
| `AGENT_API_TOKEN` | 空 | HTTP 接口 Bearer 令牌，留空不鉴权（仅限本地） |
| `AGENT_ADDR` | `127.0.0.1:8095` | HTTP 服务监听地址 |
| `AGENT_WORKSPACE` | `./workspace` | 文件工具的工作区根目录 |
| `AGENT_MEMORY_DIR` | `./memory` | 记忆存储目录 |
| `AGENT_SKILLS_DIR` | `./skills` | Skill 规范目录 |
| `AGENT_MCP_ENABLED` | `true` | 是否接入 MCP Server |
| `MCP_COMMAND` | `npx` | 启动 MCP Server 的命令 |
| `MCP_ARGS` | 内置 Playwright 参数 | MCP 启动参数（留空用默认） |
| `MCP_NODE_PATH` | 自动探测 | 可选，`node.exe` 绝对路径 |
| `MCP_NPX_CLI_PATH` | 自动探测 | 可选，`npx-cli.js` 绝对路径 |

## 测试

```powershell
go test ./...
```

## 工作原理（简述）

1. `main.go` 读取配置，创建工具注册表，注册本地 `file`、`shell` 工具。
2. 若开启 MCP，`mcp.Manager` 启动 Playwright 子进程、完成握手、发现浏览器工具，经适配器注册进同一注册表。
3. 组装模型客户端、记忆仓库、Skill 提示词，创建 `Agent`。
4. 每轮对话：拼接系统提示 + 历史 + 用户输入，把注册表工具定义发给模型；模型返回工具调用时执行并回灌结果，循环直到无工具调用，最终回复落盘记忆。
5. HTTP 入口把请求转给 `Agent.Chat`；TUI 入口在终端内直接调用同一 `Agent`。

## 说明

- `workspace/`、`memory/`、`.env`、截图与 MCP 运行时快照均已在 `.gitignore` 中忽略，不会进入版本库。
- MCP 返回的图片内容仅以占位文本记录，不保存图片数据。
