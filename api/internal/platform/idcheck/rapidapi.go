package idcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// rapidAPIHost is the RapidAPI ID Game Checker host (also sent as x-rapidapi-host).
const rapidAPIHost = "id-game-checker.p.rapidapi.com"

// rapidAPIVerifier resolves player IDs via RapidAPI's ID Game Checker
// (GET https://{host}/{game-slug}/{id}). Auth is the x-rapidapi-key header.
type rapidAPIVerifier struct {
	key     string
	baseURL string // override point for tests; defaults to https://{rapidAPIHost}
	http    *http.Client
}

// newRapidAPIVerifier builds a RapidAPI-backed Verifier, erroring if the key is
// missing.
func newRapidAPIVerifier(key string) (Verifier, error) {
	if key == "" {
		return nil, fmt.Errorf("rapidapi: x-rapidapi-key is required")
	}
	return &rapidAPIVerifier{
		key:     key,
		baseURL: "https://" + rapidAPIHost,
		http:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// rapidAPIResponse is the ID Game Checker reply. A not-found lookup returns
// {error:true, status:404, msg:"id_not_found"} over HTTP 200, so the JSON body —
// not the transport status — is authoritative. data.is_ban is absent for some
// games (decodes to false).
type rapidAPIResponse struct {
	Error bool   `json:"error"`
	Msg   string `json:"msg"`
	Data  struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		IsBan    bool   `json:"is_ban"`
	} `json:"data"`
}

// Verify looks up playerID under the game slug. It returns ErrIDNotFound when the
// provider reports the id does not exist, and a wrapped error on any transport,
// auth/quota, or unexpected-shape failure (the caller fails open on those).
// playerID is a sensitive input and is never logged here.
func (v *rapidAPIVerifier) Verify(ctx context.Context, slug, playerID string) (Account, error) {
	endpoint := fmt.Sprintf("%s/%s/%s", v.baseURL, url.PathEscape(slug), url.PathEscape(playerID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Account{}, err
	}
	req.Header.Set("x-rapidapi-host", rapidAPIHost)
	req.Header.Set("x-rapidapi-key", v.key)

	resp, err := v.http.Do(req)
	if err != nil {
		return Account{}, fmt.Errorf("rapidapi request: %w", err)
	}
	defer resp.Body.Close()

	// RapidAPI returns 4xx/5xx for auth/quota problems (distinct from the in-body
	// id_not_found, which arrives as HTTP 200). Treat those as availability errors
	// so the caller fails open rather than blocking a sale on our own outage.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Account{}, fmt.Errorf("rapidapi status %d", resp.StatusCode)
	}

	var parsed rapidAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Account{}, fmt.Errorf("rapidapi decode: %w", err)
	}
	if parsed.Msg == "id_not_found" || (parsed.Error && parsed.Data.Username == "") {
		return Account{}, ErrIDNotFound
	}
	if parsed.Error || parsed.Data.Username == "" {
		return Account{}, fmt.Errorf("rapidapi: unexpected response (msg=%q)", parsed.Msg)
	}
	return Account{Username: parsed.Data.Username, Banned: parsed.Data.IsBan}, nil
}
