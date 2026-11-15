package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// azureOpenAI calls the Azure OpenAI v1 chat-completions API
// (POST {endpoint}/openai/v1/chat/completions, no api-version needed) with the
// api-key header. The deployment name is sent as the model.
type azureOpenAI struct {
	endpoint   string
	key        string
	deployment string
	http       *http.Client
}

func newAzureOpenAI(endpoint, key, deployment string) (Completer, error) {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" || key == "" || deployment == "" {
		return nil, errors.New("azureopenai: endpoint, API key and deployment are required")
	}
	return &azureOpenAI{
		endpoint:   endpoint,
		key:        key,
		deployment: deployment,
		http:       &http.Client{Timeout: 25 * time.Second},
	}, nil
}

type aoaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aoaiJSONSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type aoaiResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema *aoaiJSONSchema `json:"json_schema,omitempty"`
}

type aoaiRequest struct {
	Model               string              `json:"model"`
	Messages            []aoaiMessage       `json:"messages"`
	MaxCompletionTokens int                 `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string              `json:"reasoning_effort,omitempty"`
	ResponseFormat      *aoaiResponseFormat `json:"response_format,omitempty"`
}

type aoaiResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content string `json:"content"`
			Refusal string `json:"refusal"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete implements [Completer]. A content-filter block (HTTP 400
// code=content_filter, or finish_reason=content_filter) maps to [ErrFiltered];
// a truncated reply (finish_reason=length) and any other failure are errors.
// Prompt and completion text are never logged here.
func (a *azureOpenAI) Complete(ctx context.Context, req Request) (Response, error) {
	body := aoaiRequest{
		Model:               a.deployment,
		MaxCompletionTokens: req.MaxOutputTokens,
		// Minimal reasoning keeps latency and output-token cost low; the task is
		// grounded lookup, not multi-step reasoning. (gpt-5-family models take no
		// temperature.)
		ReasoningEffort: "minimal",
	}
	if req.System != "" {
		body.Messages = append(body.Messages, aoaiMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, aoaiMessage(m))
	}
	if req.Schema != nil {
		name := req.SchemaName
		if name == "" {
			name = "response"
		}
		body.ResponseFormat = &aoaiResponseFormat{
			Type:       "json_schema",
			JSONSchema: &aoaiJSONSchema{Name: name, Strict: true, Schema: req.Schema},
		}
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint+"/openai/v1/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", a.key)

	res, err := a.http.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("azureopenai: request failed: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Response{}, fmt.Errorf("azureopenai: read body: %w", err)
	}
	var out aoaiResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Response{}, fmt.Errorf("azureopenai: HTTP %d, unparseable body", res.StatusCode)
	}
	if res.StatusCode != http.StatusOK {
		if out.Error != nil && out.Error.Code == "content_filter" {
			return Response{}, ErrFiltered
		}
		code := ""
		if out.Error != nil {
			code = out.Error.Code
		}
		return Response{}, fmt.Errorf("azureopenai: HTTP %d (%s)", res.StatusCode, code)
	}
	if len(out.Choices) == 0 {
		return Response{}, errors.New("azureopenai: no choices in response")
	}
	c := out.Choices[0]
	usage := Response{
		InputTokens:  out.Usage.PromptTokens,
		CachedTokens: out.Usage.PromptTokensDetails.CachedTokens,
		OutputTokens: out.Usage.CompletionTokens,
	}
	switch c.FinishReason {
	case "content_filter":
		return usage, ErrFiltered
	case "length":
		return usage, errors.New("azureopenai: completion truncated (max tokens)")
	}
	if c.Message.Refusal != "" {
		return usage, ErrFiltered
	}
	usage.Content = c.Message.Content
	return usage, nil
}
