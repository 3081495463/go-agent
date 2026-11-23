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

type Conversation struct {
	ID       string        `json:"id"`
	Summary  string        `json:"summary"`
	Messages []llm.Message `json:"messages"`
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
