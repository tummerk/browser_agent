package llm

import (
	"browser-agent/internal/entity"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
)

const SystemPrompt = `Ты — автономный браузерный агент. Твоя цель — эффективно управлять браузером.

### ПРОТОКОЛ РАБОТЫ:
1. Анализируй DOM.
2. Планируй действия.
3. Выполняй действия через инструменты.
4. В конце вызови "submit_task_result".

### ⚡ СУПЕР-СПОСОБНОСТЬ: МАССОВЫЕ ДЕЙСТВИЯ (BATCHING)
Ты умеешь выполнять несколько действий за один ответ. Это делает тебя в 10 раз быстрее!

**✅ КОГДА ГРУППИРОВАТЬ:**
- Удаление нескольких элементов (чекбоксы).
- Заполнение большой формы (ввод имени, потом фамилии, потом email).
- Последовательность: [type(1), type(2), click(3)].

**⛔ КОГДА ГРУППИРОВАТЬ НЕЛЬЗЯ (ОПАСНОСТЬ!):**
- Если действие меняет URL или обновляет страницу (переход по ссылке, кнопка "Поиск", кнопка "Войти").
- **ПРАВИЛО:** Действие, меняющее страницу, должно быть **ЕДИНСТВЕННЫМ** или **ПОСЛЕДНИМ** в списке.

### ФОРМАТ ОТВЕТА:
- НЕ пиши "Я нажимаю 5 кнопок". Сразу возвращай массив tool_calls:
  [click(10), click(11), click(12), click(13)]

### ВАЖНО:
- Не пиши "Я закончил" текстом. Используй только инструмент "submit_task_result".
- ID элементов меняются после перезагрузки.
`

// Это чистая функция: вход -> выход. Её легко тестировать.
// ConstructMessages создает полную цепочку сообщений для отправки в LL
func ConstructMessages(task string, history []entity.ActionRecord, state *entity.BrowserState) []openai.ChatCompletionMessageParamUnion {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(SystemPrompt),
	}

	// --- HISTORY BLOCK (JSON Style) ---
	// Мы меняем формат на JSONL (JSON Lines). Это стандартный формат для логов машин.
	// Модель поймет контекст, но НЕ будет пытаться генерировать такой текст в ответе,
	// так как она знает, что ее выход - это Tool Calls.
	if len(history) > 0 {
		var historyBuilder strings.Builder
		historyBuilder.WriteString("PREVIOUS ACTIONS LOG (Read-Only Context):\n")

		for i, record := range history {
			// Создаем временную структуру для чистого JSON
			logEntry := map[string]interface{}{
				"step":    i + 1,
				"thought": record.Reasoning,
				"action":  record.Action,
				"args":    record.Args, // Это уже строка JSON, но для лога пойдет
				"result":  record.Result,
			}

			// Маршалим в строку
			jsonBytes, _ := json.Marshal(logEntry)
			historyBuilder.WriteString(string(jsonBytes) + "\n")
		}

		messages = append(messages, openai.UserMessage(historyBuilder.String()))
	}

	// --- CURRENT TASK & STATE ---
	userContent := fmt.Sprintf(
		"CURRENT TASK: %s\n\n"+
			"CURRENT BROWSER STATE:\n"+
			"URL: %s\n"+
			"Title: %s\n\n"+
			"DOM STRUCTURE (Interactive Elements):\n%s",
		task,
		state.URL,
		state.Title,
		state.DOMSummary,
	)
	messages = append(messages, openai.UserMessage(userContent))

	return messages
}
