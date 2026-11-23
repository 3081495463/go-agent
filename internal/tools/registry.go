package tools

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
)

// Registry 管理 Agent 可以使用的工具
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry 创建工具注册中心
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// isNilTool 判断工具是否为 nil，包括接口底层包装了 nil 指针的情况。
func isNilTool(tool Tool) bool {
	if tool == nil {
		return true
	}

	v := reflect.ValueOf(tool)

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface,
		reflect.Map, reflect.Slice,
		reflect.Func, reflect.Chan:
		return v.IsNil()
	default:
		return false
	}
}

// Register 注册工具
func (r *Registry) Register(tool Tool) error {
	if r == nil {
		return fmt.Errorf("registry is nil")
	}

	if isNilTool(tool) {
		return fmt.Errorf("tool is nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tools == nil {
		r.tools = make(map[string]Tool)
	}

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// Get 根据名称获取工具
func (r *Registry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List 获取所有工具名称
func (r *Registry) List() []string {
	if r == nil {
		return []string{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))

	for name := range r.tools {
		names = append(names, name)
	}

	// 保证返回结果顺序稳定
	sort.Strings(names)

	return names
}
