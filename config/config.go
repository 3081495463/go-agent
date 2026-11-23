package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Config 聚合运行所需的配置，全部来自环境变量。
type Config struct {
	APIKey    string
	Model     string
	BaseURL   string
	APIToken  string
	Addr      string
	Workspace string
	MemoryDir string
	SkillsDir string

	// MCP 开关：默认开启，接入 Playwright MCP。
	MCPEnabled bool
	// MCPCommand 启动 MCP Server 的命令（默认 npx）。
	MCPCommand string
	// MCPArgs 启动参数（默认 Playwright MCP 有头模式，使用自带 Chromium）。
	MCPArgs []string
	// MCPNodePath / MCPNPXCLIPath：可选的 node.exe 与 npx-cli.js 绝对路径，
	// 在自动探测不可靠时显式指定，优先级最高。
	MCPNodePath    string
	MCPNPXCLIPath string
}

// 默认 Playwright 自带 Chromium 可执行文件（本机缓存路径）。
const defaultChromiumPath = `C:\Users\Lenovo\AppData\Local\ms-playwright\chromium-1255\chrome-win64\chrome.exe`

// Load 读取环境变量，缺失项使用默认值。
// 启动前先尝试从项目根目录的 .env 加载（已存在的环境变量优先，不被覆盖）。
func Load() Config {
	loadDotEnv(".env")

	cfg := Config{
		APIKey:    os.Getenv("DEEPSEEK_API_KEY"),
		Model:     getEnv("DEEPSEEK_MODEL", "deepseek-flash"),
		BaseURL:   getEnv("DEEPSEEK_BASE_URL", "https://tokenrhythm.studio/v1"),
		APIToken:  os.Getenv("AGENT_API_TOKEN"),
		Addr:      getEnv("AGENT_ADDR", "127.0.0.1:8095"),
		Workspace: getEnv("AGENT_WORKSPACE", "./workspace"),
		MemoryDir: getEnv("AGENT_MEMORY_DIR", "./memory"),
		SkillsDir: getEnv("AGENT_SKILLS_DIR", "./skills"),

		MCPEnabled: getEnv("AGENT_MCP_ENABLED", "true") == "true",
		MCPCommand: getEnv("MCP_COMMAND", "npx"),
		MCPNodePath:    os.Getenv("MCP_NODE_PATH"),
		MCPNPXCLIPath:  os.Getenv("MCP_NPX_CLI_PATH"),
	}

	if raw := os.Getenv("MCP_ARGS"); strings.TrimSpace(raw) != "" {
		cfg.MCPArgs = splitArgs(raw)
	} else {
		cfg.MCPArgs = defaultMCPArgs()
	}

	return cfg
}

// defaultMCPArgs 给出 Playwright MCP 的默认启动参数。
// 默认有头（headed）模式，浏览器窗口可见；如需要无头可在 MCP_ARGS 里加 --headless。
func defaultMCPArgs() []string {
	return []string{
		"@playwright/mcp@latest",
		"--executable-path", defaultChromiumPath,
	}
}

// splitArgs 按空白拆分命令行参数；忽略首尾空白。
func splitArgs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv 读取指定路径的 .env 文件并写入环境变量。
// 规则：# 开头为注释；KEY=VALUE；双引号值支持转义；已存在的环境变量不覆盖。
// 文件不存在或解析失败都静默跳过，不影响程序启动。
func loadDotEnv(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 去掉行内注释（仅当 # 前是空白且不在引号内时，这里做简单处理）。
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = parseEnvValue(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		// 已有环境变量优先，不覆盖。
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

// parseEnvValue 去掉值两侧的引号并处理简单转义。
func parseEnvValue(v string) string {
	if len(v) >= 2 {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			inner := v[1 : len(v)-1]
			if first == '"' {
				inner = strings.ReplaceAll(inner, `\n`, "\n")
				inner = strings.ReplaceAll(inner, `\"`, `"`)
				inner = strings.ReplaceAll(inner, `\\`, `\`)
			}
			return inner
		}
	}
	// 去掉行内尾随注释。
	if idx := strings.Index(v, " #"); idx >= 0 {
		v = strings.TrimSpace(v[:idx])
	}
	return v
}
