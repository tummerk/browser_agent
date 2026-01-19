package llm

import (
	"browser-agent/internal/entity"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
)

func ParseResponse(msg openai.ChatCompletionMessage) ([]entity.ToolCall, error) {
	var toolCalls []entity.ToolCall

	// 1. Извлекаем ход мыслей (Reasoning).
	// Qwen обычно пишет мысли в Content перед вызовом тулзов.
	reasoning := msg.Content

	// 2. Если тулзов нет, возможно модель просто болтает или просит уточнения
	if len(msg.ToolCalls) == 0 {
		return nil, nil
	}

	// 3. Проходим по всем вызовам инструментов
	for _, tc := range msg.ToolCalls {
		// Создаем твою структуру
		myCall := entity.ToolCall{
			Name:      tc.Function.Name,
			Reasoning: reasoning, // Привязываем общую мысль к действию
			Args:      make(map[string]interface{}),
		}

		// Парсим JSON-аргументы (они приходят строкой)
		err := json.Unmarshal([]byte(tc.Function.Arguments), &myCall.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to parse arguments for %s: %w", tc.Function.Name, err)
		}

		toolCalls = append(toolCalls, myCall)
	}

	return toolCalls, nil
}
