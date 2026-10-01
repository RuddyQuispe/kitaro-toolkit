package main

import (
	"context"
	"fmt"

	"kitaro-toolkit/internal/config"
	"kitaro-toolkit/internal/tools"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// RunTool executes a single-input tool (formatters, validators) by ID.
func (a *App) RunTool(toolID string, input string, opts map[string]string) (string, error) {
	t := tools.Get(toolID)
	if t == nil {
		return "", fmt.Errorf("unknown tool: %s", toolID)
	}
	return t.Run(input, opts)
}

// RunDiffTool executes a two-input comparator (JSON/XML diff) by ID.
func (a *App) RunDiffTool(toolID string, left string, right string) (string, error) {
	t := tools.GetDiff(toolID)
	if t == nil {
		return "", fmt.Errorf("unknown diff tool: %s", toolID)
	}
	return t.Run(left, right)
}

// GetConfig returns the persisted user config (theme, prefs).
func (a *App) GetConfig() (config.Config, error) {
	return config.Load()
}

// SetTheme persists the user's theme choice ("dark" or "light").
func (a *App) SetTheme(theme string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Theme = theme
	return config.Save(cfg)
}
