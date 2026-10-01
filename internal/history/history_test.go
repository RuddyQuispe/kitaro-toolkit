package history

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestStore opens a Store in a fresh temp dir and closes it when the test
// ends, so the SQLite file handle is released before TempDir cleanup.
func newTestStore(t *testing.T, dir string) *Store {
	t.Helper()
	s := NewWithDir(dir)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestStore_AddAndList_NewestFirst(t *testing.T) {
	s := newTestStore(t, t.TempDir())

	if err := s.Add(Entry{ToolID: "jsonfmt", ToolName: "JSON Formatter", Kind: "run", Input: "a", Output: "A"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(Entry{ToolID: "xmlfmt", ToolName: "XML Formatter", Kind: "run", Input: "b", Output: "B"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ToolID != "xmlfmt" || entries[1].ToolID != "jsonfmt" {
		t.Fatalf("expected newest-first order, got %+v", entries)
	}
	for _, e := range entries {
		if e.ID == "" {
			t.Errorf("expected generated ID, got empty for %+v", e)
		}
		if e.CreatedAt.IsZero() {
			t.Errorf("expected generated CreatedAt, got zero for %+v", e)
		}
	}
}

func TestMaxEntries_Is100(t *testing.T) {
	if MaxEntries != 100 {
		t.Fatalf("expected MaxEntries to be 100, got %d", MaxEntries)
	}
}

func TestStore_Add_EvictsOldestBeyondMaxEntries(t *testing.T) {
	s := newTestStore(t, t.TempDir())

	const extra = 5
	for i := 0; i < MaxEntries+extra; i++ {
		if err := s.Add(Entry{ToolID: "jsonfmt", Output: fmt.Sprintf("run-%d", i)}); err != nil {
			t.Fatalf("Add #%d: %v", i, err)
		}
	}

	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != MaxEntries {
		t.Fatalf("expected history capped at %d entries, got %d", MaxEntries, len(entries))
	}
	// Queue semantics: the first `extra` runs were evicted, newest is first.
	if got, want := entries[0].OutputPreview, fmt.Sprintf("run-%d", MaxEntries+extra-1); got != want {
		t.Errorf("expected newest entry %q first, got %q", want, got)
	}
	if got, want := entries[len(entries)-1].OutputPreview, fmt.Sprintf("run-%d", extra); got != want {
		t.Errorf("expected oldest kept entry %q last, got %q", want, got)
	}
}

func TestStore_Add_PreservesAllFields(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	in := Entry{ToolID: "xmldiff", ToolName: "XML Diff", Kind: "diff", Left: "<a/>", Right: "<b/>", Output: "-<a/>\n+<b/>"}
	if err := s.Add(in); err != nil {
		t.Fatalf("Add: %v", err)
	}
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got, err := s.Get(entries[0].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ToolID != in.ToolID || got.ToolName != in.ToolName || got.Kind != in.Kind ||
		got.Left != in.Left || got.Right != in.Right || got.Output != in.Output || got.Input != "" {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, in)
	}
}

func TestStore_Delete(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	_ = s.Add(Entry{ToolID: "jsonfmt", Output: "A"})
	_ = s.Add(Entry{ToolID: "xmlfmt", Output: "B"})

	entries, _ := s.List()
	toDelete := entries[0].ID

	if err := s.Delete(toDelete); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after delete, got %d", len(entries))
	}
	if entries[0].ID == toDelete {
		t.Fatalf("deleted entry still present")
	}
}

func TestStore_Clear(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	_ = s.Add(Entry{ToolID: "jsonfmt", Output: "A"})
	_ = s.Add(Entry{ToolID: "xmlfmt", Output: "B"})

	if err := s.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestStore_PersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()

	s1 := newTestStore(t, dir)
	_ = s1.Add(Entry{ToolID: "jsonfmt", Output: "A"})

	s2 := newTestStore(t, dir)
	entries, err := s2.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 || entries[0].ToolID != "jsonfmt" {
		t.Fatalf("expected entry to persist across store instances, got %+v", entries)
	}
}

func TestStore_MigratesLegacyJSON(t *testing.T) {
	dir := t.TempDir()
	older := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	legacy := map[string][]Entry{"entries": {
		{ID: "1", ToolID: "jsonfmt", ToolName: "JSON Formatter", Kind: "run", Input: "a", Output: "A", CreatedAt: older},
		{ID: "2", ToolID: "xmldiff", ToolName: "XML Diff", Kind: "diff", Left: "l", Right: "r", Output: "d", CreatedAt: newer},
	}}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "history.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	s := newTestStore(t, dir)
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 migrated entries, got %d", len(entries))
	}
	if entries[0].ToolID != "xmldiff" || entries[1].ToolID != "jsonfmt" {
		t.Fatalf("expected migrated entries newest first, got %+v", entries)
	}
	newest, err := s.Get(entries[0].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	oldest, err := s.Get(entries[1].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !oldest.CreatedAt.Equal(older) || newest.Left != "l" || oldest.Input != "a" {
		t.Fatalf("expected migrated fields preserved, got %+v / %+v", newest, oldest)
	}

	if _, err := os.Stat(filepath.Join(dir, "history.json")); !os.IsNotExist(err) {
		t.Errorf("expected history.json to be renamed away, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "history.json.bak")); err != nil {
		t.Errorf("expected history.json.bak to exist: %v", err)
	}

	// A new run after migration is queued behind the imported ones.
	if err := s.Add(Entry{ToolID: "sqldiff", Output: "x"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	entries, _ = s.List()
	if entries[0].ToolID != "sqldiff" {
		t.Errorf("expected new run to be newest, got %+v", entries[0])
	}
}

func TestStore_UsesSQLiteFile(t *testing.T) {
	dir := t.TempDir()
	s := newTestStore(t, dir)
	if err := s.Add(Entry{ToolID: "jsonfmt", Output: "A"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "history.db")); err != nil {
		t.Fatalf("expected history.db to exist: %v", err)
	}
}

func TestStore_Options_RoundTrip(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	opts := map[string]string{"mode": "minify", "dialect": "plsql"}
	if err := s.Add(Entry{ToolID: "jsonfmt", Kind: "run", Input: "{}", Output: "{}", Options: opts}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got, err := s.Get(entries[0].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Options) != 2 || got.Options["mode"] != "minify" || got.Options["dialect"] != "plsql" {
		t.Fatalf("expected options round trip, got %+v", got.Options)
	}
}

func TestStore_Get_ReturnsFullEntry(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	_ = s.Add(Entry{ToolID: "jsonfmt", ToolName: "JSON Formatter", Kind: "run", Input: "in-1", Output: "out-1"})
	_ = s.Add(Entry{ToolID: "xmlfmt", ToolName: "XML Formatter", Kind: "run", Input: "in-2", Output: "out-2"})

	entries, _ := s.List()
	got, err := s.Get(entries[1].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != entries[1].ID || got.ToolID != "jsonfmt" || got.ToolName != "JSON Formatter" ||
		got.Input != "in-1" || got.Output != "out-1" || got.CreatedAt.IsZero() {
		t.Fatalf("unexpected entry: %+v", got)
	}
	if got.Options != nil {
		t.Errorf("expected nil options for an entry saved without any, got %+v", got.Options)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	for _, id := range []string{"999", "not-a-number"} {
		if _, err := s.Get(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q): expected ErrNotFound, got %v", id, err)
		}
	}
}

func TestStore_List_ReturnsPreviewsWithoutBodies(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	long := strings.Repeat("abc  \n\t ", 200)
	_ = s.Add(Entry{ToolID: "jsonfmt", Kind: "run", Input: long, Output: "short   out", Options: map[string]string{"mode": "pretty"}})
	_ = s.Add(Entry{ToolID: "jsondiff", Kind: "diff", Left: "l  \n l", Right: "r", Output: "d"})

	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, e := range entries {
		if e.Input != "" || e.Left != "" || e.Right != "" || e.Output != "" || e.Options != nil {
			t.Errorf("expected List to omit full bodies, got %+v", e)
		}
	}
	diff, run := entries[0], entries[1]
	if diff.LeftPreview != "l l" || diff.RightPreview != "r" || diff.OutputPreview != "d" {
		t.Errorf("unexpected diff previews: %+v", diff)
	}
	if run.OutputPreview != "short out" {
		t.Errorf("expected collapsed output preview, got %q", run.OutputPreview)
	}
	if n := len([]rune(run.InputPreview)); n != PreviewLength+1 || !strings.HasSuffix(run.InputPreview, "…") {
		t.Errorf("expected input preview truncated to %d runes plus ellipsis, got %d: %q", PreviewLength, n, run.InputPreview)
	}
	if strings.Contains(run.InputPreview, "  ") || strings.ContainsAny(run.InputPreview, "\n\t") {
		t.Errorf("expected whitespace collapsed, got %q", run.InputPreview)
	}
}

func TestStore_MigratesOldSchema_AddsOptionsColumn(t *testing.T) {
	dir := t.TempDir()
	old, err := sql.Open("sqlite", filepath.Join(dir, "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = old.Exec(`
CREATE TABLE entries (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	tool_id    TEXT NOT NULL,
	tool_name  TEXT NOT NULL,
	kind       TEXT NOT NULL,
	input      TEXT NOT NULL DEFAULT '',
	left_text  TEXT NOT NULL DEFAULT '',
	right_text TEXT NOT NULL DEFAULT '',
	output     TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL
);
INSERT INTO entries (tool_id, tool_name, kind, input, output, created_at) VALUES ('jsonfmt', 'JSON Formatter', 'run', 'a', 'A', 1);`)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s := newTestStore(t, dir)
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected the old row to survive migration, got %+v", entries)
	}
	got, err := s.Get(entries[0].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Input != "a" || got.Output != "A" || got.Options != nil {
		t.Fatalf("unexpected migrated row: %+v", got)
	}
	if err := s.Add(Entry{ToolID: "jsonfmt", Output: "B", Options: map[string]string{"mode": "minify"}}); err != nil {
		t.Fatalf("Add after migration: %v", err)
	}

	// Reopening an already-migrated DB must not try to add the column again.
	_ = s.Close()
	s2 := newTestStore(t, dir)
	entries, err = s2.List()
	if err != nil {
		t.Fatalf("List after reopen: %v", err)
	}
	newest, err := s2.Get(entries[0].ID)
	if err != nil || newest.Options["mode"] != "minify" {
		t.Fatalf("expected options on post-migration entry, got %+v (err %v)", newest, err)
	}
}
