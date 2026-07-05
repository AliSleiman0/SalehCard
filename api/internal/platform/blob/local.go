package blob

import (
	"context"
	"fmt"
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
