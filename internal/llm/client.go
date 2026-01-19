package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"browser-agent/internal/entity"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Client реализует интерфейс Brain
type Client struct {
	client *openai.Client
	model  string

	Task          string
	ActionHistory []entity.ActionRecord
}

// New создает новый экземпляр LLM клиента
func New(apiKey, model, baseURL string) *Client {
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	// Для OpenRouter/Groq/LocalLLM важно менять BaseURL
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(opts...)
	return &Client{
		client:        &client,
		model:         model,
		ActionHistory: []entity.ActionRecord{},
	}
}

// Reset сбрасывает состояние мозга (для новой задачи)
func (c *Client) Reset() {
	c.Task = ""
	c.ActionHistory = []entity.ActionRecord{}
}

// RecordAction сохраняет результат выполнения действия в историю.
// Теперь принимает entity.ToolCall целиком, что удобнее.
func (c *Client) RecordAction(call entity.ToolCall, result string) {
	// Превращаем map аргументов обратно в JSON строку для истории,
	// чтобы LLM видела, с какими именно параметрами она вызывала функцию.
	argsBytes, _ := json.Marshal(call.Args)

	c.ActionHistory = append(c.ActionHistory, entity.ActionRecord{
		Reasoning: call.Reasoning,
		Action:    call.Name,
		Args:      string(argsBytes),
		Result:    result,
	})
}

// Step принимает текущее состояние браузера и возвращает список действий (ToolCalls)
func (c *Client) Step(ctx context.Context, state *entity.BrowserState, task string) ([]entity.ToolCall, error) {
	// 1. Если задача пришла впервые, запоминаем её
	if c.Task == "" && task != "" {
		c.Task = task
	}

	// 2. Формируем контекст сообщений (System + History + Current DOM)
	// Используем функцию ConstructMessages из prompt.go
	messages := ConstructMessages(c.Task, c.ActionHistory, state)

	// 3. Отправляем запрос в LLM
	// Обрати внимание: используем openai.F() для обертки параметров
	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       c.model,
		Messages:    messages,
		Tools:       defineTools(),            // Твоя функция определения тулзов
		Temperature: openai.Opt[float64](0.1), // Правильный хелпер для float64
		// ToolChoice: не указываем, по умолчанию "auto"
	})

	if err != nil {
		return nil, fmt.Errorf("llm request failed: %w", err)
	}

	// 4. Парсим ответ
	msg := resp.Choices[0].Message
	return parseResponseToEntity(msg)
}

// --- Вспомогательные функции ---

// parseResponseToEntity конвертирует ответ SDK в твои структуры entity.ToolCall
func parseResponseToEntity(msg openai.ChatCompletionMessage) ([]entity.ToolCall, error) {
	// Если тулзов нет, но есть текст - выводим его в лог (для дебага)
	if len(msg.ToolCalls) == 0 {
		fmt.Printf("🤖 Agent Reasoning (No Tools): %s\n", msg.Content)
		return nil, nil
	}

	var result []entity.ToolCall
	reasoning := msg.Content // Мысли агента перед вызовом (CoT)

	for _, tc := range msg.ToolCalls {
		var args map[string]interface{}

		// Unmarshal аргументов JSON
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			// Если модель вернула битый JSON, возвращаем ошибку
			return nil, fmt.Errorf("failed to parse tool arguments for %s: %w", tc.Function.Name, err)
		}

		// ВАЖНО: JSON числа приходят как float64.
		// Для удобства сразу конвертируем ID в int, так как в entity.ToolCall.Args мы чаще всего ждем int.
		if idVal, ok := args["id"]; ok {
			if f, ok := idVal.(float64); ok {
				args["id"] = int(f)
			}
		}

		result = append(result, entity.ToolCall{
			Name:      tc.Function.Name,
			Args:      args,
			Reasoning: reasoning, // Прикрепляем общую мысль к каждому действию в пачке
		})
	}

	return result, nil
}
