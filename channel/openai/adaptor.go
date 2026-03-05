package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"relay/apitype"
	"relay/channel"
	"relay/model"
)

func init() {
	channel.Register(channel.Provider{
		Name:       "OpenAI",
		APIType:    apitype.OpenAI,
		EnvKey:     "OPENAI_API_KEY",
		BaseURLEnv: "OPENAI_BASE_URL",
		DefaultURL: "https://api.openai.com",
		Models:     ModelList,
		NewAdaptor: func() channel.Adaptor { return &Adaptor{} },
	})
}

type Adaptor struct {
	meta *channel.Meta
}

func (a *Adaptor) Init(meta *channel.Meta) {
	a.meta = meta
}

func (a *Adaptor) GetChannelName() string { return "OpenAI" }

func (a *Adaptor) GetModelList() []string { return ModelList }

func (a *Adaptor) GetRequestURL(meta *channel.Meta) (string, error) {
	return strings.TrimRight(meta.BaseURL, "/") + "/v1/chat/completions", nil
}

func (a *Adaptor) SetupRequestHeader(req *http.Request, meta *channel.Meta) error {
	channel.SetupCommonRequestHeader(req, meta)
	req.Header.Set("Authorization", "Bearer "+meta.APIKey)
	return nil
}

func (a *Adaptor) ConvertRequest(req *model.ChatRequest) (any, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.Stream {
		if req.StreamOptions == nil {
			req.StreamOptions = &model.StreamOptions{}
		}
		req.StreamOptions.IncludeUsage = true
	}
	return req, nil
}

func (a *Adaptor) DoRequest(ctx context.Context, meta *channel.Meta, body io.Reader) (*http.Response, error) {
	return channel.DoRequestHelper(a, ctx, meta, body)
}

func (a *Adaptor) DoResponse(_ context.Context, w io.Writer, resp *http.Response, meta *channel.Meta) (*model.Usage, error) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream error (HTTP %d): %s", resp.StatusCode, body)
	}

	if meta.IsStream {
		return StreamHandler(w, resp)
	}
	return Handler(w, resp)
}

// Handler 处理非流式响应。
func Handler(w io.Writer, resp *http.Response) (*model.Usage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var chatResp model.ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	for _, choice := range chatResp.Choices {
		if choice.Message != nil {
			if content, ok := choice.Message.Content.(string); ok {
				fmt.Fprint(w, content)
			}
		}
	}

	return chatResp.Usage, nil
}

// StreamHandler 处理 SSE 流式响应。
func StreamHandler(w io.Writer, resp *http.Response) (*model.Usage, error) {
	var usage *model.Usage
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[len("data: "):]
		if data == "[DONE]" {
			break
		}

		var chunk model.StreamResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		for _, choice := range chunk.Choices {
			if choice.Delta != nil {
				if content, ok := choice.Delta.Content.(string); ok {
					fmt.Fprint(w, content)
				}
			}
		}

		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}

	return usage, scanner.Err()
}
