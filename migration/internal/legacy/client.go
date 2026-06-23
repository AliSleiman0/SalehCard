package legacy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DefaultBaseURL is the legacy storefront API root.
const DefaultBaseURL = "https://api.salehcard.com"

// Client fetches legacy categories, caching every raw response under RawDir so
// re-runs are offline and deterministic (the cache also serves as test fixtures).
type Client struct {
	BaseURL string
	RawDir  string
	Delay   time.Duration // politeness delay before each network fetch
	Refresh bool          // when true, bypass the cache and re-fetch
	HTTP    *http.Client
}

// NewClient builds a Client with sensible defaults.
func NewClient(rawDir string, delay time.Duration, refresh bool) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		RawDir:  rawDir,
		Delay:   delay,
		Refresh: refresh,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// rawPath is the on-disk cache path for one (category, lang) response.
func (c *Client) rawPath(id int, lang string) string {
	return filepath.Join(c.RawDir, fmt.Sprintf("category_%d_%s.json", id, lang))
}

// FetchCategory returns the parsed body for one category in one language,
// reading the on-disk cache when present (unless Refresh is set) and otherwise
// fetching over HTTP and writing the raw bytes to the cache.
func (c *Client) FetchCategory(id int, lang string) (*CategoryData, error) {
	path := c.rawPath(id, lang)

	if !c.Refresh {
		if b, err := os.ReadFile(path); err == nil {
			return parseCategory(b)
		}
	}

	if c.Delay > 0 {
		time.Sleep(c.Delay)
	}
	url := fmt.Sprintf("%s/api/category/products/%d?lang=%s", c.BaseURL, id, lang)
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch category %d (%s): %w", id, lang, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch category %d (%s): status %d", id, lang, resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.RawDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return nil, err
	}
	return parseCategory(b)
}

// Fetched is one visited category: its identity, parent, and full body.
type Fetched struct {
	ID       int
	ParentID int // 0 for a root
	Name     string
	Meta     *Subcategory // the parent's index entry for this child; nil for roots
	Data     *CategoryData
}

// Walk fetches rootID and recurses through every sub-category discovered in each
// response (no hard-coded sub-ids), returning one Fetched per visited category.
// A visited set guards against cycles / duplicate edges.
func (c *Client) Walk(rootID int, lang string) ([]Fetched, error) {
	visited := map[int]bool{}
	var out []Fetched

	type frame struct {
		id     int
		parent int
		name   string
		meta   *Subcategory
	}
	stack := []frame{{id: rootID, parent: 0, name: ""}}

	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[f.id] {
			continue
		}
		visited[f.id] = true

		data, err := c.FetchCategory(f.id, lang)
		if err != nil {
			return nil, err
		}
		name := f.name
		if name == "" {
			name = data.Name
		}
		out = append(out, Fetched{ID: f.id, ParentID: f.parent, Name: name, Meta: f.meta, Data: data})

		for i := range data.Categories {
			sub := data.Categories[i]
			if !visited[sub.ID] {
				stack = append(stack, frame{id: sub.ID, parent: f.id, name: sub.Name, meta: &data.Categories[i]})
			}
		}
	}
	return out, nil
}

func parseCategory(b []byte) (*CategoryData, error) {
	var r CategoryResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("parse category response: %w", err)
	}
	return &r.Data, nil
}
