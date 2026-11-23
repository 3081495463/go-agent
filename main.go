package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"go-agent/internal/agent"
	"go-agent/internal/llm"
	"go-agent/internal/mcp"
	"go-agent/internal/memory"
	"go-agent/internal/server"
	"go-agent/internal/skills"
	"go-agent/internal/tools"
	"go-agent/internal/tui"

	"go-agent/config"
)

func main() {
	// 子命令：默认起 HTTP 服务；`tui` 打开终端聊天界面。
	if len(os.Args) > 1 && os.Args[1] == "tui" {
		runTUI()
		return
	}

	cfg := config.Load()

	if cfg.APIKey == "" {
		log.Fatal("请先配置 DEEPSEEK_API_KEY")
	}

	myAgent, mcpManager, err := buildAgent(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer mcpManager.Close()

	// HTTP 服务（供前端调用）
	api := server.New(myAgent, cfg.APIToken)

	log.Printf("Agent workspace: %s", filepath.Clean(cfg.Workspace))
	log.Printf("工具: %v", myAgent.ListTools())
	log.Printf("监听 %s", cfg.Addr)

	log.Fatal(api.Listen(cfg.Addr))
}

// buildAgent 按配置组装工具注册表、MCP、模型、记忆与技能，返回可用的 Agent。
// 同时返回 MCP 管理器，调用方负责在结束时 Close。
func buildAgent(cfg config.Config) (*agent.Agent, *mcp.Manager, error) {
	// 工作区与记忆目录
	if err := os.MkdirAll(cfg.Workspace, 0o755); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(cfg.MemoryDir, 0o755); err != nil {
		return nil, nil, err
	}

	// 工具注册表
	registry := tools.NewRegistry()

	if err := registry.Register(tools.NewFileTool(cfg.Workspace)); err != nil {
		return nil, nil, err
	}
	if err := registry.Register(tools.NewShellTool(cfg.Workspace)); err != nil {
		return nil, nil, err
	}

	// MCP Server（默认接入 Playwright MCP，stdio 子进程）
	mcpManager := mcp.NewManager()

	if cfg.MCPEnabled {
		ctx := context.Background()
		if err := mcpManager.AddServer(ctx, registry, mcp.ServerConfig{
			Name:       "playwright",
			Command:    cfg.MCPCommand,
			Args:       cfg.MCPArgs,
			NodePath:   cfg.MCPNodePath,
			NPXCLIPath: cfg.MCPNPXCLIPath,
		}); err != nil {
			mcpManager.Close()
			return nil, nil, err
		}
	}

	// 模型（DeepSeek 或兼容 OpenAI 接口的中转服务）
	model, err := llm.NewDeepseekClient(cfg.APIKey, cfg.Model, cfg.BaseURL)
	if err != nil {
		mcpManager.Close()
		return nil, nil, err
	}

	// 长期记忆
	store := memory.NewStore(cfg.MemoryDir)

	// 加载 Skill 规范，拼进系统提示词
	skillPrompt := ""
	loadedSkills, err := skills.Load(cfg.SkillsDir)
	if err != nil {
		log.Printf("警告: 加载 Skill 失败: %v", err)
	} else {
		skillPrompt = skills.BuildSkillPrompt(loadedSkills)
	}

	// Agent 组装
	myAgent := agent.NewAgent(model, registry)
	myAgent.Store = store
	myAgent.SkillPrompt = skillPrompt

	return myAgent, mcpManager, nil
}

// runTUI 启动 Bubble Tea 终端聊天界面。
func runTUI() {
	cfg := config.Load()
	if cfg.APIKey == "" {
		log.Fatal("请先配置 DEEPSEEK_API_KEY")
	}

	myAgent, mcpManager, err := buildAgent(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer mcpManager.Close()

	// 默认会话 ID，保证多轮对话能读到长期记忆。
	session := "tui-session"
	if err := tui.Run(myAgent, session); err != nil {
		log.Fatal(err)
	}
}
