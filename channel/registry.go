package channel

// Provider 描述一个 AI 厂商的注册信息，由各适配器包在 init() 中注册。
type Provider struct {
	Name       string
	APIType    int
	EnvKey     string // API Key 对应的环境变量名
	BaseURLEnv string // 可选：用于覆盖 BaseURL 的环境变量名
	DefaultURL string
	Models     []string
	NewAdaptor func() Adaptor
}

var providers []Provider

// Register 将一个 Provider 加入全局注册表。
func Register(p Provider) {
	providers = append(providers, p)
}

// Providers 返回所有已注册的 Provider。
func Providers() []Provider {
	return providers
}

// GetAdaptor 根据 API 类型从注册表中创建对应的适配器实例。
func GetAdaptor(apiType int) Adaptor {
	for _, p := range providers {
		if p.APIType == apiType {
			return p.NewAdaptor()
		}
	}
	return nil
}
