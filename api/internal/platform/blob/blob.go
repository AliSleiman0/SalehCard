// Package blob is the outbound object-storage port for product image uploads,
// together with its provider adapters (Azure Blob, local disk). Callers depend
// only on the [Storage] port; [New] selects an adapter from [Config] so the
// storage backend swaps purely by configuration (ports & adapters / hexagonal),
// mirroring platform/sms. Adding a provider is a new adapter file plus one case
// in [New].
package blob

import (
	"context"
	"fmt"
)

// Storage is the outbound port: it persists data under key and returns a
// publicly-reachable URL.
type Storage interface {
	Upload(ctx context.Context, key, contentType string, data []byte) (url string, err error)
	// Delete removes the object stored under key. It is idempotent: deleting
	// an object that does not exist returns nil, so retries and double-deletes
	// are always safe.
	Delete(ctx context.Context, key string) error
}

// Config selects and configures the active storage adapter.
type Config struct {
	// Provider is one of "azure" or "local" (default "local").
	Provider string
	Azure    AzureConfig
	Local    LocalConfig
}

// AzureConfig holds Azure Blob Storage connection settings.
type AzureConfig struct {
	ConnectionString string
	Container        string
}

// LocalConfig holds the dev local-disk adapter settings.
type LocalConfig struct {
	// Dir is the filesystem directory files are written to (default "./uploads").
	Dir string
	// PublicBase is the URL prefix the local adapter builds public URLs from
	// (e.g. "http://localhost:8090").
	PublicBase string
}

// New builds the [Storage] for cfg.Provider. An empty or "local" provider
// returns the dev [LocalStorage]; an unknown provider, or a selected provider
// missing required credentials, returns an error.
func New(cfg Config) (Storage, error) {
	switch cfg.Provider {
	case "", "local":
		return newLocalStorage(cfg.Local), nil
	case "azure":
		return newAzureStorage(cfg.Azure)
	default:
		return nil, fmt.Errorf("unknown storage provider %q", cfg.Provider)
	}
}
