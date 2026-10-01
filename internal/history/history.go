// Package history persists a small, capped log of tool runs (formats and
// diffs) so the user can revisit past results without re-running the tool.
// It behaves as a FIFO queue of the most recent MaxEntries runs — each new
// run evicts the oldest once the cap is reached. This is a recent-activity
// log, not an audit trail. Storage is a SQLite file (pure-Go driver, no CGO)
// so each Add is a small transaction instead of a full-file rewrite.
package history

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"kitaro-toolkit/internal/config"

	_ "modernc.org/sqlite"
)

// MaxEntries caps how many history records are kept. Adding beyond this
// drops the oldest entries first.
const MaxEntries = 100

// PreviewLength is how many characters (runes) of each body List returns,
// after whitespace is collapsed. Longer bodies end with "…".
const PreviewLength = 160

// ErrNotFound is returned by Get when no entry has the given ID.
var ErrNotFound = errors.New("history entry not found")

// Entry is one recorded tool execution. Left/Right are only set for diff
// tools; Input is only set for single-input tools. Options holds the tool
// options the run used (e.g. formatter mode).
//
// Get returns the full bodies; List returns only metadata plus the *Preview
// fields, leaving Input/Left/Right/Output/Options empty.
type Entry struct {
	ID            string            `json:"id"`
	ToolID        string            `json:"toolId"`
	ToolName      string            `json:"toolName"`
	Kind          string            `json:"kind"` // "run" or "diff"
	Input         string            `json:"input,omitempty"`
	Left          string            `json:"left,omitempty"`
	Right         string            `json:"right,omitempty"`
	Output        string            `json:"output"`
	Options       map[string]string `json:"options,omitempty"`
	InputPreview  string            `json:"inputPreview,omitempty"`
	LeftPreview   string            `json:"leftPreview,omitempty"`
	RightPreview  string            `json:"rightPreview,omitempty"`
	OutputPreview string            `json:"outputPreview,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

// Store reads and writes history in dir/history.db. Access is serialized
// with a mutex (and a single DB connection) since Wails may invoke bound
// methods concurrently.
type Store struct {
	mu      sync.Mutex
	dir     string
	db      *sql.DB
	openErr error
}

// New returns a Store backed by the user's config directory, matching where
// internal/config keeps the app's other persisted state.
func New() (*Store, error) {
	dir, err := config.AppDir()
	if err != nil {
		return nil, err
	}
	s := NewWithDir(dir)
	return s, s.openErr
}

// NewWithDir returns a Store backed by an explicit directory — used by tests
// to avoid touching the real user config directory. If the database cannot
// be opened, every later call returns that error instead of panicking.
func NewWithDir(dir string) *Store {
	s := &Store{dir: dir}
	s.db, s.openErr = s.open()
	return s
}

const schema = `
CREATE TABLE IF NOT EXISTS entries (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	tool_id    TEXT NOT NULL,
	tool_name  TEXT NOT NULL,
	kind       TEXT NOT NULL,
	input      TEXT NOT NULL DEFAULT '',
	left_text  TEXT NOT NULL DEFAULT '',
	right_text TEXT NOT NULL DEFAULT '',
	output     TEXT NOT NULL DEFAULT '',
	options    TEXT NOT NULL DEFAULT '{}',
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS entries_created_at ON entries(created_at);
`

func (s *Store) open() (*sql.DB, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, err
	}
	dsn := filepath.Join(s.dir, "history.db") +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	if err := addOptionsColumn(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrateLegacyJSON(db, s.dir); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// addOptionsColumn upgrades databases created before entries had an
// options column. CREATE TABLE IF NOT EXISTS leaves such tables untouched.
func addOptionsColumn(db *sql.DB) error {
	rows, err := db.Query(`SELECT name FROM pragma_table_info('entries')`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		if name == "options" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()
	_, err = db.Exec(`ALTER TABLE entries ADD COLUMN options TEXT NOT NULL DEFAULT '{}'`)
	return err
}

// migrateLegacyJSON imports the pre-SQLite history.json (if present), oldest
// first so AUTOINCREMENT ids keep queue order, then renames it to
// history.json.bak so the import runs only once.
func migrateLegacyJSON(db *sql.DB, dir string) error {
	legacyPath := filepath.Join(dir, "history.json")
	data, err := os.ReadFile(legacyPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var legacy struct {
		Entries []Entry `json:"entries"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("read legacy history.json: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, e := range legacy.Entries {
		if e.CreatedAt.IsZero() {
			e.CreatedAt = time.Now()
		}
		if err := insert(tx, e); err != nil {
			return err
		}
	}
	if err := trim(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return os.Rename(legacyPath, legacyPath+".bak")
}

func insert(tx *sql.Tx, e Entry) error {
	opts := "{}"
	if len(e.Options) > 0 {
		data, err := json.Marshal(e.Options)
		if err != nil {
			return err
		}
		opts = string(data)
	}
	_, err := tx.Exec(
		`INSERT INTO entries (tool_id, tool_name, kind, input, left_text, right_text, output, options, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ToolID, e.ToolName, e.Kind, e.Input, e.Left, e.Right, e.Output, opts, e.CreatedAt.UnixNano(),
	)
	return err
}

// preview collapses whitespace runs to single spaces and truncates to
// PreviewLength runes.
func preview(text string) string {
	collapsed := strings.Join(strings.Fields(text), " ")
	runes := []rune(collapsed)
	if len(runes) > PreviewLength {
		return string(runes[:PreviewLength]) + "…"
	}
	return collapsed
}

// trim keeps only the newest MaxEntries rows. Ordering by id (not
// created_at) guarantees strict insertion order even on clock ties.
func trim(tx *sql.Tx) error {
	_, err := tx.Exec(
		`DELETE FROM entries WHERE id NOT IN (SELECT id FROM entries ORDER BY id DESC LIMIT ?)`,
		MaxEntries,
	)
	return err
}

// Add records a new entry, assigning it an ID and timestamp, and evicts the
// oldest entries beyond MaxEntries in the same transaction.
func (s *Store) Add(e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.openErr != nil {
		return s.openErr
	}

	e.CreatedAt = time.Now()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := insert(tx, e); err != nil {
		return err
	}
	if err := trim(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// previewChars bounds how much of each body SQLite hands back for List: a
// wide margin over PreviewLength so indentation-heavy bodies (pretty
// JSON/XML) still collapse to a full preview.
const previewChars = 1000

// List returns all entries, newest first, as lightweight summaries: the
// metadata and *Preview fields only. Use Get for an entry's full content.
func (s *Store) List() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.openErr != nil {
		return nil, s.openErr
	}

	rows, err := s.db.Query(
		`SELECT id, tool_id, tool_name, kind,
		        substr(input, 1, ?1), substr(left_text, 1, ?1), substr(right_text, 1, ?1), substr(output, 1, ?1),
		        length(input) > ?1, length(left_text) > ?1, length(right_text) > ?1, length(output) > ?1,
		        created_at
		 FROM entries ORDER BY id DESC`,
		previewChars,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var (
			e                          Entry
			id                         int64
			createdAt                  int64
			input, left, right, output string
			cut                        [4]bool
		)
		if err := rows.Scan(&id, &e.ToolID, &e.ToolName, &e.Kind,
			&input, &left, &right, &output,
			&cut[0], &cut[1], &cut[2], &cut[3],
			&createdAt); err != nil {
			return nil, err
		}
		e.ID = strconv.FormatInt(id, 10)
		e.CreatedAt = time.Unix(0, createdAt)
		e.InputPreview = previewOf(input, cut[0])
		e.LeftPreview = previewOf(left, cut[1])
		e.RightPreview = previewOf(right, cut[2])
		e.OutputPreview = previewOf(output, cut[3])
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// previewOf previews a body that SQLite may have cut at previewChars; a cut
// body always gets the ellipsis even if it collapses below PreviewLength.
func previewOf(text string, cut bool) string {
	p := preview(text)
	if cut && !strings.HasSuffix(p, "…") {
		p += "…"
	}
	return p
}

// Get returns one entry with its full content and options, or ErrNotFound.
func (s *Store) Get(id string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.openErr != nil {
		return Entry{}, s.openErr
	}

	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return Entry{}, ErrNotFound // not an ID this store could have issued
	}

	var (
		e         Entry
		opts      string
		createdAt int64
	)
	err = s.db.QueryRow(
		`SELECT tool_id, tool_name, kind, input, left_text, right_text, output, options, created_at
		 FROM entries WHERE id = ?`, n,
	).Scan(&e.ToolID, &e.ToolName, &e.Kind, &e.Input, &e.Left, &e.Right, &e.Output, &opts, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}
	if err := json.Unmarshal([]byte(opts), &e.Options); err != nil {
		return Entry{}, fmt.Errorf("decode options of entry %s: %w", id, err)
	}
	if len(e.Options) == 0 {
		e.Options = nil
	}
	e.ID = strconv.FormatInt(n, 10)
	e.CreatedAt = time.Unix(0, createdAt)
	return e, nil
}

// Delete removes a single entry by ID. Deleting an unknown ID is a no-op.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.openErr != nil {
		return s.openErr
	}

	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil // not an ID this store could have issued
	}
	_, err = s.db.Exec(`DELETE FROM entries WHERE id = ?`, n)
	return err
}

// Clear empties the history.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.openErr != nil {
		return s.openErr
	}

	_, err := s.db.Exec(`DELETE FROM entries`)
	return err
}

// Close releases the database handle. Safe to call on a Store that failed
// to open.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	s.openErr = errors.New("history store is closed")
	return err
}
