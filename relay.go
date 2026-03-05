package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"relay/channel"
	"relay/model"
)

// Relay 根据请求的模型名选择渠道，将请求转发到上游并将响应写入 output。
func Relay(ctx context.Context, channels []*channel.Config, req *model.ChatRequest, output io.Writer) (*model.Usage, error) {
	ch := matchChannel(channels, req.Model)
	if ch == nil {
		return nil, fmt.Errorf("no channel for model %q", req.Model)
	}

	meta := ch.BuildMeta(req.Model, req.Stream)

	adaptor := channel.GetAdaptor(meta.APIType)
	if adaptor == nil {
		return nil, fmt.Errorf("unsupported api type: %d", meta.APIType)
	}
	adaptor.Init(meta)

	converted, err := adaptor.ConvertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("convert request: %w", err)
	}
	body, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := adaptor.DoRequest(ctx, meta, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	usage, err := adaptor.DoResponse(ctx, output, resp, meta)
	if err != nil {
		return nil, fmt.Errorf("do response: %w", err)
	}

	return usage, nil
}

func matchChannel(channels []*channel.Config, modelName string) *channel.Config {
	for _, ch := range channels {
		for _, m := range ch.Models {
			if m == modelName || m == "*" {
				return ch
			}
		}
	}
	return nil
}
