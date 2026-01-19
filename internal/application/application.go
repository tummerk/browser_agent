package application

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"browser-agent/internal/agent"
	"browser-agent/internal/browser"
	"browser-agent/internal/config" // Импортируем твой пакет конфига
	"browser-agent/internal/llm"
)

func Run(ctx context.Context) error {
	// 1. Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		// LoadConfig возвращает понятную ошибку, если нет ключа
		return fmt.Errorf("initialization failed: %w", err)
	}

	log.Println("🚀 Инициализация системы...")
	log.Printf("🔧 Конфигурация: Model=%s, BaseURL=%s", cfg.Model, cfg.Url)

	// 2. Запускаем браузер (Persistent Session)
	log.Println("🔌 Запускаем браузер...")
	// false = headless выключен (мы видим браузер), true = скрытый режим
	browserSvc, err := browser.NewBrowserService(ctx, false)
	if err != nil {
		return fmt.Errorf("browser launch error: %w", err)
	}
	defer browserSvc.Close()

	// 3. Поднимаем Мозг (LLM) используя данные из конфига
	llmClient := llm.New(
		cfg.APIKey,
		cfg.Model,
		cfg.Url,
	)

	// 4. Создаем Оркестратора (Агента)
	orchestrator := agent.New(browserSvc, llmClient)

	// 5. Запускаем REPL цикл (Read-Eval-Print Loop)
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n==================================================")
	fmt.Println("🤖 AGENT ONLINE. Браузер готов к командам.")
	fmt.Println("   (Введите 'exit', 'quit' или Ctrl+C для выхода)")
	fmt.Println("==================================================")

	for {
		// Проверка на отмену контекста (graceful shutdown)
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fmt.Print("\n💬 Введите новую задачу > ")
		task, err := reader.ReadString('\n')
		if err != nil {
			break // EOF
		}

		task = strings.TrimSpace(task)

		if task == "" {
			continue
		}
		if task == "exit" || task == "quit" {
			log.Println("👋 Завершение работы...")
			break
		}

		log.Printf("🏁 [START] Выполняю задачу: '%s'", task)

		// Запускаем задачу через Агента
		orchestrator.RunTask(task)

		log.Println("✨ Задача завершена (или прервана). Готов к следующей.")
	}

	return nil
}
