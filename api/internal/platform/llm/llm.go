// Package llm is the large-language-model completion port together with its
// provider adapters (Azure OpenAI, stub). Callers depend only on the
// [Completer] port; [New] selects an adapter from [Config] so providers swap
// purely by configuration (ports & adapters, like platform/sms and
// platform/idcheck). Adding a provider is a new adapter file plus one case in
// [New] — nothing outside this package changes.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrFiltered is returned when the provider's content-safety layer blocked the
// prompt (e.g. a jailbreak attempt) or the completion. Callers answer with a
// polite refusal rather than an error.
var ErrFiltered = errors.New("llm: content filtered")

// Message is one chat turn. Role is "user" or "assistant" (the system prompt
// travels separately in [Request.System]).
type Message struct {
	Role    string
	Content string
}

// Request is one completion call.
type Request struct {
	// System is the system prompt (instructions + any grounding material).
	System   string
	Messages []Message
	// MaxOutputTokens caps the completion length (the main cost lever).
	MaxOutputTokens int
	// SchemaName + Schema request a strict JSON-schema structured output; with a
	// nil Schema the reply is free text.
	SchemaName string
	Schema     json.RawMessage
}

// Response is the completion text plus token usage (for cost accounting).
type Response struct {
	Content      string
	InputTokens  int
	CachedTokens int
	OutputTokens int
}

// Completer is the outbound port: it runs one chat completion.
type Completer interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

// Config selects and configures the active adapter.
type Config struct {
	// Provider is "azureopenai" or "stub" (default "stub").
	Provider string
	// Azure OpenAI: the resource endpoint (https://<name>.openai.azure.com), the
	// API key (server-side only) and the model deployment name.
	AzureEndpoint   string
	AzureAPIKey     string
	AzureDeployment string
}

// New builds the [Completer] for cfg.Provider. An empty or "stub" provider
// returns [Stub]; "azureopenai" requires endpoint, key and deployment; an
// unknown provider errors.
func New(cfg Config) (Completer, error) {
	switch cfg.Provider {
	case "", "stub":
		return Stub{}, nil
	case "azureopenai":
		return newAzureOpenAI(cfg.AzureEndpoint, cfg.AzureAPIKey, cfg.AzureDeployment)
	default:
		return nil, fmt.Errorf("unknown LLM provider %q", cfg.Provider)
	}
}

// Stub is the zero-cost dev adapter. With a schema it replies with an
// "unanswerable" structured answer (so callers exercise their fallback path);
// without one it replies with a fixed string. It never calls out.
type Stub struct{}

// Complete implements [Completer].
func (Stub) Complete(_ context.Context, req Request) (Response, error) {
	if req.Schema != nil {
		return Response{Content: `{"answerable":false,"answer":"","sourceIds":[]}`}, nil
	}
	return Response{Content: "stub"}, nil
}
