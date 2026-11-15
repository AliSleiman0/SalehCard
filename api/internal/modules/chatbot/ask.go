package chatbot

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"github.com/AliSleiman0/salehcard/api/internal/platform/llm"
	"github.com/AliSleiman0/salehcard/api/internal/platform/ratelimit"
)

const (
	maxAskRunes      = 500
	maxHistoryTurns  = 6 // messages (≈ the last three question/answer pairs)
	maxHistoryRunes  = 1000
	maxOutputTokens  = 700
	askModelDeadline = 30 * time.Second
)

// Errors the handler maps to HTTP statuses.
var (
	// ErrAIDisabled: the assistant is switched off (Settings) or has no live
	// provider. 503 CHATBOT_AI_DISABLED — clients fall back to the keyword bot.
	ErrAIDisabled = errors.New("ai assistant disabled")
	// ErrAILimited: a per-user / per-IP / global cap was hit. 429.
	ErrAILimited = errors.New("ai assistant rate limited")
	// ErrAIUnavailable: the limiter or the model provider failed. 503 — the
	// limiter fails CLOSED here because every call is billed.
	ErrAIUnavailable = errors.New("ai assistant unavailable")
)

// Limits are the billed-call caps.
type Limits struct {
	UserMax      int
	UserWindow   time.Duration
	IPMax        int
	IPWindow     time.Duration
	GlobalMax    int
	GlobalWindow time.Duration
}

type settingsReader interface {
	Get(ctx context.Context) (*settings.Settings, error)
}

// Assistant answers free-typed help questions from the admin's knowledge text.
type Assistant struct {
	kb       KnowledgeRepository
	logs     LogRepository
	settings settingsReader
	model    llm.Completer
	limiter  ratelimit.Limiter
	limits   Limits
	// live reports whether the configured provider can really answer (false
	// for the stub outside development).
	live bool
	now  func() time.Time
}

// NewAssistant wires the assistant.
func NewAssistant(kb KnowledgeRepository, logs LogRepository, st settingsReader, model llm.Completer,
	limiter ratelimit.Limiter, limits Limits, live bool) *Assistant {
	return &Assistant{kb: kb, logs: logs, settings: st, model: model, limiter: limiter,
		limits: limits, live: live, now: time.Now}
}

// Turn is one prior message the client sends for context.
type Turn struct {
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
}

// AskInput is POST /api/v1/chatbot/ask.
type AskInput struct {
	Question string `json:"question"`
	Lang     string `json:"lang"`
	// Platform is the client OS ("ios" | "android"). iOS never receives the
	// android-only knowledge (App Review), so the model cannot answer from it.
	Platform string `json:"platform"`
	History  []Turn `json:"history"`
}

// AskResult is the response. Answered=false means Answer is the fallback text
// and the client should surface the support contact.
type AskResult struct {
	LogID       string `json:"logId,omitempty"`
	Answer      string `json:"answer"`
	Answered    bool   `json:"answered"`
	ShowSupport bool   `json:"showSupport"`
}

// Enabled reports whether the assistant is on (Settings toggle + a live
// provider). A settings read error reports off.
func (a *Assistant) Enabled(ctx context.Context) (bool, *settings.Settings) {
	if !a.live {
		return false, nil
	}
	s, err := a.settings.Get(ctx)
	if err != nil || s == nil || !s.ChatbotAIEnabled {
		return false, s
	}
	return true, s
}

// validate normalizes the input, returning a user-facing message on failure.
func (in AskInput) validate() (AskInput, string, error) {
	out := in
	out.Question = strings.TrimSpace(in.Question)
	if out.Question == "" {
		return out, "type a question", errInvalid
	}
	if utf8.RuneCountInString(out.Question) > maxAskRunes {
		return out, "the question is too long (max 500 characters)", errInvalid
	}
	if out.Lang != "ar" {
		out.Lang = "en"
	}
	if len(in.History) > maxHistoryTurns {
		out.History = in.History[len(in.History)-maxHistoryTurns:]
	}
	hist := make([]Turn, 0, len(out.History))
	for _, t := range out.History {
		if t.Role != "user" && t.Role != "assistant" {
			return out, "invalid conversation history", errInvalid
		}
		c := strings.TrimSpace(t.Content)
		if c == "" {
			continue
		}
		if utf8.RuneCountInString(c) > maxHistoryRunes {
			c = string([]rune(c)[:maxHistoryRunes])
		}
		hist = append(hist, Turn{Role: t.Role, Content: c})
	}
	out.History = hist
	return out, "", nil
}

// allow applies the per-user, per-IP and global caps in that order, failing
// closed on a limiter error.
func (a *Assistant) allow(ctx context.Context, userID, ip string) error {
	checks := []struct {
		key    string
		max    int
		window time.Duration
	}{
		{"chatai:u:" + userID, a.limits.UserMax, a.limits.UserWindow},
		{"chatai:ip:" + ip, a.limits.IPMax, a.limits.IPWindow},
		{"chatai:global", a.limits.GlobalMax, a.limits.GlobalWindow},
	}
	for _, c := range checks {
		ok, err := a.limiter.Allow(ctx, c.key, c.max, c.window)
		if err != nil {
			slog.Warn("chatbot: rate limiter error, refusing AI call (fail closed)", "error", err)
			return ErrAIUnavailable
		}
		if !ok {
			return ErrAILimited
		}
	}
	return nil
}

// Ask answers one question. Validation failures return errInvalid with a
// message; gate/limit/provider failures return the Err* sentinels. Every call
// that reaches the model is logged to chat_logs (never to slog).
func (a *Assistant) Ask(ctx context.Context, userID, ip string, raw AskInput) (AskResult, string, error) {
	on, st := a.Enabled(ctx)
	if !on {
		return AskResult{}, "", ErrAIDisabled
	}
	in, msg, err := raw.validate()
	if err != nil {
		return AskResult{}, msg, err
	}
	if err := a.allow(ctx, userID, ip); err != nil {
		return AskResult{}, "", err
	}
	k, err := a.kb.Get(ctx)
	if err != nil {
		return AskResult{}, "", ErrAIUnavailable
	}
	secs := sectionsFor(k, in.Platform)

	fallback := AskResult{Answer: fallbackText(in.Lang, st.StoreName), ShowSupport: true}
	entry := ChatLog{ID: bson.NewObjectID(), UserID: userID, Lang: in.Lang, Question: in.Question, CreatedAt: a.now().UTC()}

	var res AskResult
	if len(secs) == 0 {
		// Nothing to ground on: don't pay for a call that must refuse.
		entry.Outcome = outcomeUnanswered
		res = fallback
	} else {
		msgs := make([]llm.Message, 0, len(in.History)+2)
		for _, t := range in.History {
			msgs = append(msgs, llm.Message{Role: t.Role, Content: t.Content})
		}
		// The app-language note goes last, after the question, so the system
		// prompt stays a byte-identical (cached) prefix across languages.
		msgs = append(msgs,
			llm.Message{Role: "user", Content: in.Question},
			llm.Message{Role: "system", Content: appLanguageNote(in.Lang)})

		mctx, cancel := context.WithTimeout(ctx, askModelDeadline)
		start := a.now()
		out, err := a.model.Complete(mctx, llm.Request{
			System:          buildSystemPrompt(st.StoreName, secs),
			Messages:        msgs,
			MaxOutputTokens: maxOutputTokens,
			SchemaName:      "help_answer",
			Schema:          answerSchema,
		})
		cancel()
		entry.LatencyMs = a.now().Sub(start).Milliseconds()
		entry.InputTokens, entry.CachedTokens, entry.OutputTokens = out.InputTokens, out.CachedTokens, out.OutputTokens

		switch {
		case errors.Is(err, llm.ErrFiltered):
			entry.Outcome = outcomeFiltered
			res = fallback
		case err != nil:
			entry.Outcome = outcomeError
			a.insertLog(ctx, &entry)
			slog.Warn("chatbot: AI provider call failed", "error", err) // no question text
			return AskResult{}, "", ErrAIUnavailable
		default:
			if answer, cited, ok := enforce(out.Content, secs); ok {
				entry.Outcome, entry.Answer = outcomeAnswered, answer
				res = AskResult{Answer: answer, Answered: true}
				for _, s := range cited {
					entry.Sources = append(entry.Sources, snippet(s.Text))
				}
			} else {
				entry.Outcome = outcomeUnanswered
				res = fallback
			}
		}
	}
	if a.insertLog(ctx, &entry) {
		res.LogID = entry.ID.Hex()
	}
	return res, "", nil
}

// insertLog is best-effort: a logging failure never fails the answer.
func (a *Assistant) insertLog(ctx context.Context, l *ChatLog) bool {
	if err := a.logs.Insert(ctx, l); err != nil {
		slog.Warn("chatbot: failed to write chat log", "error", err)
		return false
	}
	return true
}

// Report flags the caller's own logged answer (Play's AI-content policy:
// users must be able to report generated output).
func (a *Assistant) Report(ctx context.Context, userID string, logID bson.ObjectID) error {
	return a.logs.Flag(ctx, logID, userID, a.now().UTC())
}
