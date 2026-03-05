package channel

// Meta 携带单次请求的上下文信息，贯穿整个转发流程。
type Meta struct {
	ChannelType     int
	APIType         int
	APIKey          string
	BaseURL         string
	OriginModelName string
	ActualModelName string
	ModelMapping    map[string]string
	IsStream        bool
}

// Config 描述一个渠道的静态配置。
type Config struct {
	Name         string
	APIType      int
	BaseURL      string
	APIKey       string
	Models       []string
	ModelMapping map[string]string
}

// BuildMeta 根据渠道配置、请求模型和流式标识构建 Meta。
func (c *Config) BuildMeta(requestModel string, stream bool) *Meta {
	actual := requestModel
	if mapped, ok := c.ModelMapping[requestModel]; ok {
		actual = mapped
	}
	return &Meta{
		ChannelType:     c.APIType,
		APIType:         c.APIType,
		APIKey:          c.APIKey,
		BaseURL:         c.BaseURL,
		OriginModelName: requestModel,
		ActualModelName: actual,
		ModelMapping:    c.ModelMapping,
		IsStream:        stream,
	}
}
