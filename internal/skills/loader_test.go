package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkills(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n内容A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("忽略我"), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("应只加载 1 个 md 文件，实际 %d", len(list))
	}
	if list[0].Name != "a.md" {
		t.Fatalf("技能名不匹配: %s", list[0].Name)
	}

	prompt := BuildSkillPrompt(list)
	if prompt == "" {
		t.Fatal("技能提示词不应为空")
	}
}

func TestLoadMissingDir(t *testing.T) {
	list, err := Load(filepath.Join(t.TempDir(), "no-such-dir"))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatal("缺失目录应返回空列表")
	}
}
