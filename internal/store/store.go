package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Settings — only two fields survive per spec.
type Settings struct {
	ShowAppsInMenu       bool `json:"showAppsInMenu"`
	AutoUpdateEnabled    bool `json:"autoUpdateEnabled"`
	ClaudeDesktopUsed    bool `json:"claudeDesktopUsed"` // harness state, not shown in Settings UI
	OnboardingVersion    int  `json:"onboardingVersion"`
	HasCompletedFirstRun bool `json:"-"` // internal
}

const CurrentOnboardingVersion = 1

type Store struct {
	DBPath string
	mu     sync.Mutex
	db     *sql.DB
}

func defaultDBPath() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Launchpad", "db.sqlite")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Launchpad", "db.sqlite")
	default:
		return filepath.Join(os.Getenv("HOME"), ".launchpad", "db.sqlite")
	}
}

func (s *Store) ensureDB() error {
	if s.db != nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return nil
	}
	path := s.DBPath
	if path == "" {
		path = defaultDBPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	// minimal schema — only settings; chats/tools tables are NOT created
	schema := `
	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY CHECK (id=1),
		show_apps_in_menu BOOLEAN NOT NULL DEFAULT 1,
		auto_update_enabled BOOLEAN NOT NULL DEFAULT 1,
		claude_desktop_used BOOLEAN NOT NULL DEFAULT 0,
		onboarding_version INTEGER NOT NULL DEFAULT 0,
		has_completed_first_run BOOLEAN NOT NULL DEFAULT 0,
		device_id TEXT NOT NULL DEFAULT ''
	);
	INSERT OR IGNORE INTO settings (id) VALUES (1);
	CREATE TABLE IF NOT EXISTS claude_config (
		id INTEGER PRIMARY KEY CHECK (id=1),
		fable_5 TEXT NOT NULL DEFAULT '',
		opus_5 TEXT NOT NULL DEFAULT '',
		sonnet_5 TEXT NOT NULL DEFAULT '',
		haiku_4_5 TEXT NOT NULL DEFAULT '',
		sonnet_4_6 TEXT NOT NULL DEFAULT '',
		auto_mode BOOLEAN NOT NULL DEFAULT 0
	);
	INSERT OR IGNORE INTO claude_config (id) VALUES (1);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	// ensure device_id
	var deviceID string
	_ = db.QueryRow("SELECT device_id FROM settings WHERE id=1").Scan(&deviceID)
	if deviceID == "" {
		u, _ := uuid.NewV7()
		_, _ = db.Exec("UPDATE settings SET device_id=? WHERE id=1", u.String())
	}
	s.db = db
	return nil
}

func (s *Store) Settings() (Settings, error) {
	if err := s.ensureDB(); err != nil {
		return Settings{}, err
	}
	var st Settings
	var hasCompleted bool
	err := s.db.QueryRow(`SELECT show_apps_in_menu, auto_update_enabled, claude_desktop_used, onboarding_version, has_completed_first_run FROM settings WHERE id=1`).Scan(
		&st.ShowAppsInMenu, &st.AutoUpdateEnabled, &st.ClaudeDesktopUsed, &st.OnboardingVersion, &hasCompleted,
	)
	if err != nil {
		return Settings{}, err
	}
	st.HasCompletedFirstRun = hasCompleted
	return st, nil
}

func (s *Store) SetSettings(st Settings) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE settings SET show_apps_in_menu=?, auto_update_enabled=?, claude_desktop_used=?, onboarding_version=? WHERE id=1`,
		st.ShowAppsInMenu, st.AutoUpdateEnabled, st.ClaudeDesktopUsed, st.OnboardingVersion)
	return err
}

func (s *Store) HasCompletedFirstRun() (bool, error) {
	st, err := s.Settings()
	if err != nil {
		return false, err
	}
	return st.HasCompletedFirstRun, nil
}

func (s *Store) SetHasCompletedFirstRun(v bool) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec("UPDATE settings SET has_completed_first_run=? WHERE id=1", v)
	return err
}

type ClaudeConfig struct {
	Fable5   string `json:"fable_5"`
	Opus5    string `json:"opus_5"`
	Sonnet5  string `json:"sonnet_5"`
	Haiku45  string `json:"haiku_4_5"`
	Sonnet46 string `json:"sonnet_4_6"`
	AutoMode bool   `json:"autoMode"`
}

func (s *Store) ClaudeConfig() (ClaudeConfig, error) {
	if err := s.ensureDB(); err != nil {
		return ClaudeConfig{}, err
	}
	var c ClaudeConfig
	err := s.db.QueryRow(`SELECT fable_5, opus_5, sonnet_5, haiku_4_5, sonnet_4_6, auto_mode FROM claude_config WHERE id=1`).Scan(
		&c.Fable5, &c.Opus5, &c.Sonnet5, &c.Haiku45, &c.Sonnet46, &c.AutoMode)
	if err != nil {
		return ClaudeConfig{}, err
	}
	return c, nil
}

func (s *Store) SetClaudeConfig(c ClaudeConfig) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE claude_config SET fable_5=?, opus_5=?, sonnet_5=?, haiku_4_5=?, sonnet_4_6=?, auto_mode=? WHERE id=1`,
		c.Fable5, c.Opus5, c.Sonnet5, c.Haiku45, c.Sonnet46, c.AutoMode)
	return err
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
