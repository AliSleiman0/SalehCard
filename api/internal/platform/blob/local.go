package blob

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// defaultUploadsDir is used when LocalConfig.Dir is empty.
const defaultUploadsDir = "./uploads"

// LocalStorage writes uploads to the local filesystem — the dev adapter, used
// when STORAGE_PROVIDER is unset. Files are served back by the server's static
// /uploads mount (see server.Routes).
type LocalStorage struct {
	dir        string
	publicBase string
}

// NewLocal builds the dev local-disk adapter directly. It never returns an
// error, so a caller on a fallback path (Azure misconfig) can construct it
// without an error check — unlike the generic [New] switch.
func NewLocal(cfg LocalConfig) *LocalStorage {
	return newLocalStorage(cfg)
}

// newLocalStorage builds a disk-backed Storage from cfg. Dir defaults to
// [defaultUploadsDir]; PublicBase defaults to "" (relative URLs).
func newLocalStorage(cfg LocalConfig) *LocalStorage {
	dir := cfg.Dir
	if dir == "" {
		dir = defaultUploadsDir
	}
	return &LocalStorage{
		dir:        dir,
		publicBase: strings.TrimRight(cfg.PublicBase, "/"),
	}
}

// Upload writes data to <dir>/key, creating parent directories as needed, and
// returns a URL under PublicBase + "/uploads/" + key.
func (l *LocalStorage) Upload(_ context.Context, key, _ string, data []byte) (string, error) {
	path := filepath.Join(l.dir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("local storage: mkdir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("local storage: write: %w", err)
	}
	return l.publicBase + "/uploads/" + key, nil
}

// Delete removes <dir>/key. A file that does not exist is treated as success
// (idempotent per the [Storage] contract) but logged: beyond a benign retry,
// it can mean the server fell open from Azure to this adapter and is
// "deleting" blobs that really live (and survive) in the Azure container —
// worth noticing when the delete backs a privacy promise. Empty parent
// directories are left behind — harmless, the next upload reuses them.
func (l *LocalStorage) Delete(_ context.Context, key string) error {
	path := filepath.Join(l.dir, filepath.FromSlash(key))
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Warn("local storage: delete of a file this adapter never stored", "key", key)
			return nil
		}
		return fmt.Errorf("local storage: delete: %w", err)
	}
	return nil
}
