// Package channel 定义了 AI 厂商适配器的接口与公共辅助函数。
package channel

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"relay/model"
)

// Adaptor 是所有 AI 厂商适配器必须实现的接口。
type Adaptor interface {
	Init(meta *Meta)
	GetRequestURL(meta *Meta) (string, error)
	SetupRequestHeader(req *http.Request, meta *Meta) error
	ConvertRequest(req *model.ChatRequest) (any, error)
	DoRequest(ctx context.Context, meta *Meta, body io.Reader) (*http.Response, error)
	DoResponse(ctx context.Context, w io.Writer, resp *http.Response, meta *Meta) (*model.Usage, error)
	GetModelList() []string
	GetChannelName() string
}

// SetupCommonRequestHeader 设置所有渠道通用的请求头。
func SetupCommonRequestHeader(req *http.Request, meta *Meta) {
	req.Header.Set("Content-Type", "application/json")
	if meta.IsStream {
		req.Header.Set("Accept", "text/event-stream")
	}
}

// DoRequestHelper 是 DoRequest 的通用实现，组合了 GetRequestURL、
// SetupRequestHeader 和 HTTP 调用，各适配器可直接委托给此函数。
func DoRequestHelper(a Adaptor, ctx context.Context, meta *Meta, body io.Reader) (*http.Response, error) {
	fullURL, err := a.GetRequestURL(meta)
	if err != nil {
		return nil, fmt.Errorf("get request url: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	if err := a.SetupRequestHeader(req, meta); err != nil {
		return nil, fmt.Errorf("setup request header: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
