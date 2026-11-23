// Package tui 提供基于 Bubble Tea 的终端聊天界面。
// 直接复用组装好的 Agent，输入文字回车后异步调用 Agent.Chat，
// 回复渲染在对话区；不依赖 HTTP 服务端口。
package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"go-agent/internal/agent"
)

// message 表示一条对话记录。
type message struct {
	role string // user / assistant / system
	text string
}

// Model 是 Bubble Tea 应用模型。
type Model struct {
	agent   *agent.Agent
	session string
	width   int
	height  int
	msgs    []message
	input   textinput.Model
	spinner spinner.Model
	viewport viewport.Model
	loading bool
	err     error
	quitting bool
}

// New 创建 TUI 模型。session 用于长期记忆，为空则不落盘。
func New(a *agent.Agent, session string) *Model {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "输入指令，回车发送；↑/↓ 或滚轮翻页；Ctrl+C 退出"
	ti.Width = 80
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vp := viewport.New(80, 10)
	vp.MouseWheelEnabled = true

	return &Model{
		agent:   a,
		session: session,
		input:   ti,
		spinner: sp,
		viewport: vp,
		msgs: []message{
			{role: "system", text: "已连接 Agent，输入需求开始对话。"},
		},
	}
}

// chatDone 是异步聊天完成的消息。
type chatDone struct {
	reply string
	err   error
}

// Init 实现 tea.Model。
func (m *Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update 实现 tea.Model。
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = max(20, msg.Width-4)
		m.viewport.Width = msg.Width
		m.resizeViewport()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case chatDone:
		m.loading = false
		if msg.err != nil {
			m.msgs = append(m.msgs, message{role: "assistant", text: "出错: " + msg.err.Error()})
		} else {
			m.msgs = append(m.msgs, message{role: "assistant", text: msg.reply})
		}
		m.input.Reset()
		m.resizeViewport()
		m.viewport.GotoBottom()
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}
		// 翻页键交给 viewport 处理（不进入输入框）。
		switch msg.Type {
		case tea.KeyUp, tea.KeyPgUp, tea.KeyDown, tea.KeyPgDown:
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			return m, vpCmd
		}
		if m.loading {
			return m, nil
		}
		if msg.Type == tea.KeyEnter {
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			m.msgs = append(m.msgs, message{role: "user", text: text})
			m.loading = true
			m.input.Reset()
			m.resizeViewport()
			m.viewport.GotoBottom()
			return m, m.runChat(text)
		}
	}

	// 鼠标滚轮等消息先交给 viewport，再把按键交给输入框。
	var vpCmd, inCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.input, inCmd = m.input.Update(msg)
	return m, tea.Batch(vpCmd, inCmd)
}

// runChat 异步调用 Agent，返回产出 chatDone 的命令。
func (m *Model) runChat(text string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()
		reply, err := m.agent.Chat(ctx, m.session, text)
		return chatDone{reply: reply, err: err}
	}
}

// View 实现 tea.Model。
func (m *Model) View() string {
	if m.quitting {
		return "再见。\n"
	}

	status := ""
	if m.loading {
		status = m.spinner.View() + " Agent 处理中…"
	}

	footer := lipgloss.NewStyle().MarginTop(1).Render(m.input.View())

	parts := []string{m.viewport.View()}
	if status != "" {
		parts = append(parts, status)
	}
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// resizeViewport 按终端高度与是否显示状态行，计算对话区视口高度。
func (m *Model) resizeViewport() {
	// 预留：输入框 1 行 + 上边距 1 行；loading 时再多留 1 行状态行。
	reserved := 3
	if m.loading {
		reserved = 4
	}
	m.viewport.Height = max(5, m.height-reserved)
	m.syncViewport()
}

// syncViewport 把最新对话内容写入视口。
func (m *Model) syncViewport() {
	m.viewport.SetContent(m.renderHistory())
}

// renderHistory 把对话记录渲染成带角色配色的文本。
func (m *Model) renderHistory() string {
	var b strings.Builder
	for _, msg := range m.msgs {
		switch msg.role {
		case "user":
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("你: "))
			b.WriteString(wrap(msg.text, m.width-4))
			b.WriteString("\n\n")
		case "assistant":
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("Agent: "))
			b.WriteString(wrap(msg.text, m.width-4))
			b.WriteString("\n\n")
		default:
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(msg.text))
			b.WriteString("\n\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// wrap 简单按宽度折行，避免长回复撑破终端。
func wrap(text string, width int) string {
	if width <= 0 {
		width = 80
	}
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) <= width {
			out = append(out, line)
			continue
		}
		for len(line) > width {
			out = append(out, line[:width])
			line = line[width:]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// Run 启动 TUI，阻塞直到退出。
func Run(a *agent.Agent, session string) error {
	p := tea.NewProgram(
		New(a, session),
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
	)
	_, err := p.Run()
	return err
}
