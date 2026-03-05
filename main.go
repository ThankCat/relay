package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"relay/apitype"
	"relay/channel"
	"relay/channel/ali"
	"relay/model"
)

func main() {
	modelName := flag.String("model", "gpt-4o-mini", "模型名称")
	message := flag.String("message", "", "发送的消息（单次模式）")
	stream := flag.Bool("stream", false, "是否流式输出")
	chat := flag.Bool("chat", false, "交互式对话模式")
	baseURL := flag.String("base-url", "", "自定义 API 地址（默认 https://api.openai.com）")
	flag.Parse()

	apiKey := os.Getenv("OPENAI_API_KEY")
	aliKey := os.Getenv("DASHSCOPE_API_KEY")

	if apiKey == "" && aliKey == "" {
		fmt.Fprintln(os.Stderr, "错误: 请至少设置 OPENAI_API_KEY 或 DASHSCOPE_API_KEY")
		os.Exit(1)
	}

	base := "https://api.openai.com"
	if *baseURL != "" {
		base = *baseURL
	} else if env := os.Getenv("OPENAI_BASE_URL"); env != "" {
		base = env
	}

	var channels []*channel.Config

	if apiKey != "" {
		channels = append(channels, &channel.Config{
			Name:    "OpenAI",
			APIType: apitype.OpenAI,
			BaseURL: base,
			APIKey:  apiKey,
			Models:  []string{"*"},
		})
	}

	if aliKey != "" {
		channels = append(channels, &channel.Config{
			Name:    "Ali",
			APIType: apitype.Ali,
			BaseURL: "https://dashscope.aliyuncs.com/compatible-mode",
			APIKey:  aliKey,
			Models:  ali.ModelList,
		})
	}

	if *chat {
		runInteractiveChat(channels, *modelName, *stream)
	} else if *message != "" {
		runSingleMessage(channels, *modelName, *message, *stream)
	} else {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "\n请指定 -message 或 -chat")
		os.Exit(1)
	}
}

func runSingleMessage(channels []*channel.Config, modelName, message string, stream bool) {
	ctx := context.Background()
	req := &model.ChatRequest{
		Model:  modelName,
		Stream: stream,
		Messages: []model.Message{
			{Role: "user", Content: message},
		},
	}

	usage, err := Relay(ctx, channels, req, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	if usage != nil {
		fmt.Fprintf(os.Stderr, "[token] prompt=%d completion=%d total=%d\n",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	}
}

func runInteractiveChat(channels []*channel.Config, modelName string, stream bool) {
	ctx := context.Background()

	fmt.Printf("对话模式 (模型: %s, 流式: %v)\n", modelName, stream)
	fmt.Println("输入消息按回车发送, /quit 退出, /clear 清空历史")
	fmt.Println(strings.Repeat("-", 50))

	var history []model.Message
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n你: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch input {
		case "/quit":
			fmt.Println("再见！")
			return
		case "/clear":
			history = nil
			fmt.Println("[历史已清空]")
			continue
		}

		history = append(history, model.Message{Role: "user", Content: input})

		req := &model.ChatRequest{
			Model:    modelName,
			Stream:   stream,
			Messages: history,
		}

		fmt.Print("\nAI: ")
		var collector strings.Builder
		output := io.MultiWriter(os.Stdout, &collector)

		usage, err := Relay(ctx, channels, req, output)
		fmt.Println()

		if err != nil {
			fmt.Fprintf(os.Stderr, "[错误: %v]\n", err)
			history = history[:len(history)-1]
			continue
		}

		history = append(history, model.Message{Role: "assistant", Content: collector.String()})

		if usage != nil {
			fmt.Fprintf(os.Stderr, "[token: prompt=%d completion=%d total=%d]\n",
				usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
		}
	}
}
