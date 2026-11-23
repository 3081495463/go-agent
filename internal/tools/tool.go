package tools

import "context"

type Tool interface {
	// Name returns the name of the tool.
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]any) (string, error)
}
