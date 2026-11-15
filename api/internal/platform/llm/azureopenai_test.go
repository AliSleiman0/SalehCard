package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestAzure(t *testing.T, h http.HandlerFunc) Completer {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := newAzureOpenAI(srv.URL+"/", "k-123", "chat")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNew_Providers(t *testing.T) {
	if c, err := New(Config{}); err != nil || c == nil {
		t.Fatalf("empty provider should be stub, got %v %v", c, err)
	}
	if _, err := New(Config{Provider: "azureopenai"}); err == nil {
		t.Fatal("azureopenai without endpoint/key must error")
	}
	if _, err := New(Config{Provider: "nope"}); err == nil {
		t.Fatal("unknown provider must error")
	}
}

func TestAzure_RequestShapeAndUsage(t *testing.T) {
	var got map[string]any
	c := newTestAzure(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/v1/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("api-key") != "k-123" {
			t.Errorf("api-key header missing")
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"{\"a\":1}"}}],
			"usage":{"prompt_tokens":100,"completion_tokens":7,"prompt_tokens_details":{"cached_tokens":64}}}`))
	})
	res, err := c.Complete(context.Background(), Request{
		System:          "sys",
		Messages:        []Message{{Role: "user", Content: "hi"}},
		MaxOutputTokens: 300,
		SchemaName:      "answer",
		Schema:          json.RawMessage(`{"type":"object"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Content != `{"a":1}` || res.InputTokens != 100 || res.CachedTokens != 64 || res.OutputTokens != 7 {
		t.Fatalf("unexpected response %+v", res)
	}
	if got["model"] != "chat" || got["reasoning_effort"] != "minimal" || got["max_completion_tokens"] != float64(300) {
		t.Fatalf("unexpected body %v", got)
	}
	msgs := got["messages"].([]any)
	if len(msgs) != 2 || msgs[0].(map[string]any)["role"] != "system" {
		t.Fatalf("system message must come first: %v", msgs)
	}
	rf := got["response_format"].(map[string]any)
	js := rf["json_schema"].(map[string]any)
	if rf["type"] != "json_schema" || js["strict"] != true || js["name"] != "answer" {
		t.Fatalf("bad response_format %v", rf)
	}
	if _, has := got["temperature"]; has {
		t.Fatal("temperature must not be sent (unsupported by gpt-5 models)")
	}
}

func TestAzure_ContentFilterPromptBlock(t *testing.T) {
	c := newTestAzure(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"content_filter","message":"filtered"}}`))
	})
	if _, err := c.Complete(context.Background(), Request{}); !errors.Is(err, ErrFiltered) {
		t.Fatalf("want ErrFiltered, got %v", err)
	}
}

func TestAzure_FinishReasons(t *testing.T) {
	for reason, want := range map[string]error{"content_filter": ErrFiltered} {
		c := newTestAzure(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"` + reason + `","message":{"content":""}}]}`))
		})
		if _, err := c.Complete(context.Background(), Request{}); !errors.Is(err, want) {
			t.Fatalf("%s: want %v, got %v", reason, want, err)
		}
	}
	c := newTestAzure(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"length","message":{"content":"{\"a"}}]}`))
	})
	if _, err := c.Complete(context.Background(), Request{}); err == nil || errors.Is(err, ErrFiltered) {
		t.Fatalf("truncated reply must be a plain error, got %v", err)
	}
}

func TestAzure_HTTPErrors(t *testing.T) {
	c := newTestAzure(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"429","message":"slow down"}}`))
	})
	if _, err := c.Complete(context.Background(), Request{}); err == nil || errors.Is(err, ErrFiltered) {
		t.Fatalf("429 must be a plain error, got %v", err)
	}
	c = newTestAzure(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`<html>bad gateway</html>`))
	})
	if _, err := c.Complete(context.Background(), Request{}); err == nil {
		t.Fatal("non-JSON error body must error")
	}
}

func TestStub(t *testing.T) {
	res, err := Stub{}.Complete(context.Background(), Request{Schema: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Answerable bool `json:"answerable"`
	}
	if err := json.Unmarshal([]byte(res.Content), &v); err != nil || v.Answerable {
		t.Fatalf("stub structured reply should be unanswerable JSON: %q %v", res.Content, err)
	}
}
