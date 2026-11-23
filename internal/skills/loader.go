package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	Name    string
	Content string
}

func Load(dir string) ([]Skill, error) {

	entries, err := os.ReadDir(dir)

	if os.IsNotExist(err) {
		return []Skill{}, nil
	}

	if err != nil {
		return nil, err
	}

	var result []Skill
	total := 0

	for _, entry := range entries {

		if entry.IsDir() {
			continue
		}

		if strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			continue
		}

		if len(result) >= 20 {
			break
		}

		path := filepath.Join(dir, entry.Name())

		data, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("读取 Skill 失败: %w", err)
		}

		if len(data) > 32*1024 {
			continue
		}

		if total+len(data) > 128*1024 {
			break
		}

		result = append(result, Skill{
			Name:    entry.Name(),
			Content: string(data),
		})

		total += len(data)
	}

	return result, nil
}

func BuildSkillPrompt(skills []Skill) string {

	var builder strings.Builder

	for _, skill := range skills {

		builder.WriteString("\n\n")
		builder.WriteString("# Skill: ")
		builder.WriteString(skill.Name)
		builder.WriteString("\n")
		builder.WriteString(skill.Content)
	}

	return builder.String()
}
