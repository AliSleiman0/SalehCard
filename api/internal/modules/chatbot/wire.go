package chatbot

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"github.com/AliSleiman0/salehcard/api/internal/platform/llm"
	"github.com/AliSleiman0/salehcard/api/internal/platform/ratelimit"
)

// BuildAssistant constructs the AI help assistant from config and returns it
// with the name of the adapter actually in use. A misconfigured provider (e.g.
// azureopenai without a key) degrades to the stub with a warning instead of
// refusing to boot — the assistant then reports itself off outside development
// and the apps show only the support contacts.
func BuildAssistant(db *mongo.Database, cfg *config.Config) (*Assistant, string) {
	provider := cfg.LLMProvider
	model, err := llm.New(llm.Config{
		Provider:        provider,
		AzureEndpoint:   cfg.AzureOpenAIEndpoint,
		AzureAPIKey:     cfg.AzureOpenAIAPIKey,
		AzureDeployment: cfg.AzureOpenAIDeployment,
	})
	if err != nil {
		slog.Warn("chatbot: LLM provider misconfigured — AI assistant falls back to stub", "provider", provider, "error", err)
		model, provider = llm.Stub{}, "stub"
	}
	if provider == "" {
		provider = "stub"
	}
	// The stub never answers, so it only counts as "live" in development (to
	// exercise the client flow for free).
	live := provider != "stub" || cfg.Env == "development"

	// Always the Mongo limiter, even when RATE_LIMIT_PROVIDER=noop: these caps
	// are the spending ceiling, not an abuse nicety.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := ratelimit.EnsureIndexes(ctx, db); err != nil {
		slog.Warn("chatbot: failed to ensure rate-limit indexes", "error", err)
	}
	limits := Limits{
		UserMax: cfg.ChatbotAIUserMax, UserWindow: cfg.ChatbotAIUserWindow,
		IPMax: cfg.ChatbotAIIPMax, IPWindow: cfg.ChatbotAIIPWindow,
		GlobalMax: cfg.ChatbotAIGlobalMax, GlobalWindow: cfg.ChatbotAIGlobalWindow,
	}
	a := NewAssistant(NewMongoKnowledgeRepository(db), NewMongoLogRepository(db), settings.NewMongoRepository(db),
		model, ratelimit.NewMongoLimiter(db), limits, live)
	return a, provider
}
