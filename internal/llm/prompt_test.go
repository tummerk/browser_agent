package llm

import (
	"browser-agent/internal/entity"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
)

// Helper: Превращаем сообщение в JSON и достаем контент.
// Это работает для ЛЮБОГО типа сообщения (System, User, Assistant),
// потому что SDK гарантирует правильный JSON маршалинг.
func extractContent(t *testing.T, msg openai.ChatCompletionMessageParamUnion) string {
	// 1. Маршалим структуру SDK в байты JSON
	bytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// 2. Анмаршалим в простую временную структуру, чтобы достать текст
	var temp struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal(bytes, &temp); err != nil {
		t.Fatalf("Failed to unmarshal JSON content: %v", err)
	}

	return temp.Content
}

func TestConstructMessages_FirstStep(t *testing.T) {
	// Сценарий 1: Первый запуск, истории нет
	task := "Купить слона"
	history := []entity.ActionRecord{}
	state := &entity.BrowserState{
		URL:        "https://google.com",
		Title:      "Google",
		DOMSummary: "[1] <input> Search",
	}

	msgs := ConstructMessages(task, history, state)

	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs))
	}

	// Проверяем System Prompt
	sysContent := extractContent(t, msgs[0])
	// Сравниваем начало строки, чтобы не падать из-за пробелов
	if !strings.Contains(sysContent, "Ты — автономный AI-агент") {
		t.Error("System prompt mismatch")
	}

	// Проверяем User Message
	userContent := extractContent(t, msgs[1])
	t.Logf("\n--- [TEST LOG] User Message (First Step) ---\n%s\n--------------------------------------------", userContent)

	if !strings.Contains(userContent, "CURRENT TASK: Купить слона") {
		t.Error("Task missing in prompt")
	}
	if !strings.Contains(userContent, "google.com") {
		t.Error("URL missing in prompt")
	}
	if strings.Contains(userContent, "HISTORY OF ACTIONS") {
		t.Error("History should be empty on first step")
	}
}

func TestConstructMessages_WithHistory(t *testing.T) {
	// Сценарий 2: Агент уже что-то сделал
	task := "Удалить спам"
	history := []entity.ActionRecord{
		{
			Reasoning: "Вижу письмо от мамы",
			Action:    "click",
			Args:      `{"id": 15}`,
			Result:    "Success",
		},
	}
	state := &entity.BrowserState{
		URL:        "https://mail.yandex.ru",
		Title:      "Входящие",
		DOMSummary: "[1] <text> Входящие пусты",
	}

	msgs := ConstructMessages(task, history, state)

	if len(msgs) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(msgs))
	}

	// Проверяем сообщение с историей (оно должно быть вторым)
	historyContent := extractContent(t, msgs[1])
	t.Logf("\n--- [TEST LOG] History Message ---\n%s\n----------------------------------", historyContent)

	if !strings.Contains(historyContent, "HISTORY OF ACTIONS") {
		t.Error("Header missing")
	}
	if !strings.Contains(historyContent, "Вижу письмо от мамы") {
		t.Error("Reasoning missing")
	}

	// Проверяем текущее состояние (последнее сообщение)
	finalMsg := extractContent(t, msgs[2])
	if strings.Contains(finalMsg, "Вижу письмо от мамы") {
		t.Error("History leaked into current state message")
	}
	if !strings.Contains(finalMsg, "Входящие пусты") {
		t.Error("Current DOM missing")
	}
}
