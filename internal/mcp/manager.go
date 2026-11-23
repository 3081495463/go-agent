package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go-agent/internal/tools"
)

// ServerConfig 描述一个要接入的 MCP Server（stdio 子进程）。
type ServerConfig struct {
	Name    string
	Command string
	Args    []string
	// NodePath / NPXCLIPath 为可选的 node.exe、npx-cli.js 绝对路径覆盖。
	NodePath   string
	NPXCLIPath string
}

// Manager 管理若干 MCP Client：初始化握手、发现工具、把工具注册进注册表。
type Manager struct {
	mu      sync.Mutex
	clients []*Client
}

// NewManager 创建 MCP 管理器。
func NewManager() *Manager {
	return &Manager{}
}

// AddServer 启动并连接一个 MCP Server，发现其工具并适配后注册到 registry。
// 连接/发现失败时返回错误，由调用方决定是否阻断 Agent 启动。
func (m *Manager) AddServer(
	ctx context.Context,
	registry *tools.Registry,
	cfg ServerConfig,
) error {

	if m == nil {
		return fmt.Errorf("mcp manager is nil")
	}
	if cfg.Command == "" {
		return fmt.Errorf("mcp server %q 缺少启动命令", cfg.Name)
	}

	client := NewClient(cfg.Name, cfg.Command, cfg.Args)
	client.WithCommandOptions(commandOptions{
		nodePath:   cfg.NodePath,
		npxCLIPath: cfg.NPXCLIPath,
	})
	if err := client.Initialize(ctx); err != nil {
		_ = client.Close()
		return fmt.Errorf("MCP Server %q: %w", cfg.Name, err)
	}

	discovered, err := DiscoverTools(ctx, client)
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("发现 MCP 工具 %q: %w", cfg.Name, err)
	}

	for _, meta := range discovered {
		tool := NewToolAdapter(client, meta)
		if err := registry.Register(tool); err != nil {
			log.Printf("MCP 工具 %q 注册失败: %v", meta.Name, err)
			continue
		}
	}

	m.mu.Lock()
	m.clients = append(m.clients, client)
	m.mu.Unlock()

	log.Printf("MCP Server %q 已接入，发现工具 %d 个", cfg.Name, len(discovered))
	return nil
}

// Close 关闭所有 MCP 会话并终止子进程。
func (m *Manager) Close() {
	if m == nil {
		return
	}

	m.mu.Lock()
	clients := append([]*Client{}, m.clients...)
	m.clients = nil
	m.mu.Unlock()

	for _, client := range clients {
		_ = client.Close()
	}
}
