package ali

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"relay/channel"
	"relay/channel/openai"
	"relay/model"
)

type Adaptor struct {
	meta *channel.Meta
}

func (a *Adaptor) Init(meta *channel.Meta) {
	a.meta = meta
}

func (a *Adaptor) GetChannelName() string { return "ali" }

func (a *Adaptor) GetModelList() []string { return ModelList }

func (a *Adaptor) GetRequestURL(meta *channel.Meta) (string, error) {
	return strings.TrimRight(meta.BaseURL, "/") + "/v1/chat/completions", nil
}

func (a *Adaptor) SetupRequestHeader(req *http.Request, meta *channel.Meta) error {
	channel.SetupCommonRequestHeader(req, meta)
	req.Header.Set("Authorization", "Bearer "+meta.APIKey)
	if meta.IsStream {
		req.Header.Set("X-DashScope-SSE", "enable")
	}
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

	// DashScope 兼容模式的响应格式与 OpenAI 一致，复用 openai 的 Handler
	if meta.IsStream {
		return openai.StreamHandler(w, resp)
	}
	return openai.Handler(w, resp)
}
