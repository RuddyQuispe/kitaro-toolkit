package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"kitaro-toolkit/internal/config"
	"kitaro-toolkit/internal/history"
	"kitaro-toolkit/internal/tools"
)

// App struct
type App struct {
	ctx     context.Context
	history *history.Store
}

// NewApp creates a new App application struct
func NewApp() *App {
	h, err := history.New()
	if err != nil {
		// User config dir is unavailable (unusual, sandboxed env). History
		// becomes a no-op rather than a startup failure.
		h = history.NewWithDir(filepath.Join(os.TempDir(), "kitaro-toolkit"))
	}
	return &App{history: h}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is called when the app is closing; it releases the history DB.
func (a *App) shutdown(_ context.Context) {
	_ = a.history.Close()
}

// RunTool executes a single-input tool (formatters, validators) by ID.
func (a *App) RunTool(toolID string, input string, opts map[string]string) (string, error) {
	t := tools.Get(toolID)
	if t == nil {
		return "", fmt.Errorf("unknown tool: %s", toolID)
	}
	output, err := t.Run(input, opts)
	if err == nil {
		_ = a.history.Add(history.Entry{
			ToolID:   toolID,
			ToolName: t.Name,
			Kind:     "run",
			Input:    input,
			Output:   output,
			Options:  opts,
		})
	}
	return output, err
}

// RunDiffTool executes a two-input comparator (JSON/XML diff) by ID.
func (a *App) RunDiffTool(toolID string, left string, right string) (string, error) {
	t := tools.GetDiff(toolID)
	if t == nil {
		return "", fmt.Errorf("unknown diff tool: %s", toolID)
	}
	output, err := t.Run(left, right)
	if err == nil {
		_ = a.history.Add(history.Entry{
			ToolID:   toolID,
			ToolName: t.Name,
			Kind:     "diff",
			Left:     left,
			Right:    right,
			Output:   output,
		})
	}
	return output, err
}

// ListHistory returns past tool runs, newest first, capped at
// history.MaxEntries. Entries carry previews only; use GetHistoryEntry for
// full content.
func (a *App) ListHistory() ([]history.Entry, error) {
	return a.history.List()
}

// GetHistoryEntry returns one history entry with its full content and the
// options the run used.
func (a *App) GetHistoryEntry(id string) (history.Entry, error) {
	return a.history.Get(id)
}

// HistoryLimit returns how many runs the history keeps (history.MaxEntries),
// so the frontend never hardcodes the cap.
func (a *App) HistoryLimit() int {
	return history.MaxEntries
}

// DeleteHistoryEntry removes a single history entry by ID.
func (a *App) DeleteHistoryEntry(id string) error {
	return a.history.Delete(id)
}

// ClearHistory removes all history entries.
func (a *App) ClearHistory() error {
	return a.history.Clear()
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
