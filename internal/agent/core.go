package agent

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"browser-agent/internal/entity"
)

// Interfaces (дублируем для наглядности, в реальном проекте они в entity или interfaces)
type Browser interface {
	Observe() (*entity.BrowserState, error)
	Click(id int) error
	Type(id int, text string) error
	ReadText(id int) (string, error)
	Scroll(direction string) error
	Navigate(url string) error
	GoBack() error
	CloseTab() error
	PressKey(keyName string) error
	GetCurrentPageInfo() (url string, targetID string)
	Close()
}

type Brain interface {
	Reset()
	Step(ctx context.Context, state *entity.BrowserState, task string) ([]entity.ToolCall, error)
	// Используем сигнатуру из твоего последнего сообщения
	RecordAction(call entity.ToolCall, result string)
}

// Orchestrator связывает Мозг и Браузер
type Orchestrator struct {
	Browser Browser
	Brain   Brain
}

func New(b Browser, llm Brain) *Orchestrator {
	return &Orchestrator{
		Browser: b,
		Brain:   llm,
	}
}

// Start запускает интерактивный режим в терминале
func (o *Orchestrator) Start() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("🤖 Browser Agent Ready! Введите задачу (или 'exit' для выхода):")

	for {
		fmt.Print("\n>>> Введите задачу: ")
		if !scanner.Scan() {
			break
		}
		userInput := strings.TrimSpace(scanner.Text())

		if userInput == "exit" || userInput == "quit" {
			fmt.Println("Bye!")
			break
		}
		if userInput == "" {
			continue
		}

		// Запуск выполнения задачи
		o.RunTask(userInput)
	}
}

// RunTask выполняет одну конкретную задачу до победного
func (o *Orchestrator) RunTask(task string) {
	ctx := context.Background()

	// 1. Сбрасываем память мозга для новой задачи
	o.Brain.Reset()
	fmt.Printf("🎯 Принята задача: %s\n", task)

	step := 0
	maxSteps := 30 // Защита от бесконечного цикла

	for step < maxSteps {
		step++
		fmt.Printf("\n--- STEP %d ---\n", step)

		// A. OBSERVE (Глаза)
		state, err := o.Browser.Observe()
		if err != nil {
			log.Printf("❌ Ошибка наблюдения браузера: %v", err)
			return
		}
		fmt.Printf("🌍 URL: %s | Title: %s\n", state.URL, state.Title)

		// B. THINK (Мозг)
		toolCalls, err := o.Brain.Step(ctx, state, task)
		if err != nil {
			log.Printf("🧠 Ошибка LLM: %v", err)
			time.Sleep(2 * time.Second)
			continue // Пробуем еще раз
		}

		if len(toolCalls) == 0 {
			fmt.Println("🤔 Агент задумался (нет действий)...")
			time.Sleep(2 * time.Second)
			continue
		}

		// C. ACT (Руки)
		missionComplete := false

		for _, call := range toolCalls {
			fmt.Printf("💭 Reasoning: %s\n", call.Reasoning)
			fmt.Printf("⚡ Action: %s %+v\n", call.Name, call.Args)

			// Выполняем действие и получаем результат строкой
			resultStr := o.executeTool(call)

			fmt.Printf("✅ Result: %s\n", resultStr)

			// D. RECORD (Память)
			o.Brain.RecordAction(call, resultStr)

			// Если задача выполнена - прерываем цикл
			if call.Name == "submit_task_result" {
				missionComplete = true
			}

			switch call.Name {
			case "click", "press":
				// Если это массив действий, делаем паузу маленькой
				if len(toolCalls) > 1 {
					time.Sleep(100 * time.Millisecond) // 0.1 сек (быстро прокликиваем)
				} else {
					time.Sleep(2 * time.Second) // Одиночный клик может быть навигацией
				}

			case "type":
				time.Sleep(50 * time.Millisecond)

			case "navigate":
				time.Sleep(3 * time.Second) // Тут точно ждем
			}
		}

		if missionComplete {
			fmt.Println("\n🎉 ЗАДАЧА ВЫПОЛНЕНА! Готов к следующей.")
			break
		}
	}

	if step >= maxSteps {
		fmt.Println("⚠️ Превышен лимит шагов. Остановка.")
	}
}

// executeTool маршрутизирует вызов к методам браузера
func (o *Orchestrator) executeTool(call entity.ToolCall) string {
	var err error
	var output string = "Success"

	switch call.Name {
	case "click":
		if id, ok := getInt(call.Args, "id"); ok {
			err = o.Browser.Click(id)
		} else {
			err = fmt.Errorf("missing or invalid 'id'")
		}

	case "type":
		id, okId := getInt(call.Args, "id")
		text, okText := getString(call.Args, "text")
		if okId && okText {
			err = o.Browser.Type(id, text)
		} else {
			err = fmt.Errorf("missing 'id' or 'text'")
		}

	case "scroll":
		if dir, ok := getString(call.Args, "direction"); ok {
			err = o.Browser.Scroll(dir)
		} else {
			// Дефолт
			err = o.Browser.Scroll("down")
		}

	case "navigate":
		if url, ok := getString(call.Args, "url"); ok {
			err = o.Browser.Navigate(url)
		} else {
			err = fmt.Errorf("missing 'url'")
		}

	case "press":
		if key, ok := getString(call.Args, "key"); ok {
			err = o.Browser.PressKey(key)
		} else {
			err = fmt.Errorf("missing 'key'")
		}

	case "go_back":
		err = o.Browser.GoBack()

	case "memorize":
		if info, ok := getString(call.Args, "info"); ok {
			return fmt.Sprintf("Saved to memory: %s", info)
		}
		return "Saved info."

	case "done", "submit_task_result": // Ловим оба имени
		// Перебираем варианты ключей (приоритет final_report)
		answer := ""
		if v, ok := getString(call.Args, "final_report"); ok {
			answer = v
		} else if v, ok := getString(call.Args, "answer"); ok {
			answer = v
		} else if v, ok := getString(call.Args, "result"); ok {
			answer = v
		}

		if answer != "" {
			return fmt.Sprintf("DONE: %s", answer)
		}
		return "Task completed."

	default:
		return fmt.Sprintf("Error: Unknown tool '%s'", call.Name)
	}

	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return output
}

// --- Хелперы для безопасного извлечения типов из map[string]interface{} ---

func getInt(args map[string]interface{}, key string) (int, bool) {
	val, ok := args[key]
	if !ok || val == nil {
		return 0, false
	}
	// 1. Стандартный JSON (float64)
	if f, ok := val.(float64); ok {
		return int(f), true
	}
	// 2. Int (если вдруг)
	if i, ok := val.(int); ok {
		return i, true
	}
	// 3. String (Самое важное!)
	if s, ok := val.(string); ok {
		// Пробуем распарсить "123" или "123.0"
		if i, err := strconv.Atoi(s); err == nil {
			return i, true
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int(f), true
		}
	}
	return 0, false
}

func getString(args map[string]interface{}, key string) (string, bool) {
	val, ok := args[key]
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}
