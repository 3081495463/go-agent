package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"go-agent/internal/llm"
)

// idPattern 只允许字母、数字、下划线、短横线，杜绝路径穿越。
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Conversation 是一次会话的持久化结构。
//
// 存储分两层：
//   - Notes：跨会话可复用的关键信息（用户偏好、常用约定、关键事实），
//     由调用方按需写入，长期保留、体量小。
//   - Messages：近期对话的精简记录，只保留最近若干条，用于衔接上下文；
//     不再整段堆积每轮的长回复。
type Conversation struct {
	ID       string        `json:"id"`
	Notes    []Note        `json:"notes,omitempty"`
	Messages []llm.Message `json:"messages"`
}

// Note 是一条可复用的关键信息。
type Note struct {
	Key     string `json:"key"`     // 去重键，例如 "style"、"project"
	Content string `json:"content"` // 偏好或事实内容
}

type Store struct {
	Dir string
	mu  sync.Mutex
}

func NewStore(dir string) *Store {
	return &Store{
		Dir: dir,
	}
}

// safePath 校验 ID 并返回对应文件路径，拒绝非法 ID。
func (s *Store) safePath(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("invalid conversation id: %q", id)
	}
	return filepath.Join(s.Dir, id+".json"), nil
}

func (s *Store) Save(c Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.safePath(c.ID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.Dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("保存记忆失败: %w", err)
	}

	return nil
}

func (s *Store) Load(id string) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.safePath(id)
	if err != nil {
		return Conversation{}, err
	}

	data, err := os.ReadFile(path)

	if os.IsNotExist(err) {
		return Conversation{ID: id}, nil
	}

	if err != nil {
		return Conversation{}, err
	}

	var c Conversation

	if err := json.Unmarshal(data, &c); err != nil {
		return Conversation{}, err
	}

	return c, nil
}

// UpsertNote 写入或更新一条关键信息（按 Key 去重）。
// Notes 上限由调用方控制；超出时不新增。
func (c *Conversation) UpsertNote(key, content string) {
	if key == "" || content == "" {
		return
	}
	for i := range c.Notes {
		if c.Notes[i].Key == key {
			c.Notes[i].Content = content
			return
		}
	}
	c.Notes = append(c.Notes, Note{Key: key, Content: content})
}
