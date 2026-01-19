package main

import (
	"browser-agent/internal/browser"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	// ⚠️ НЕ ЗАБУДЬ ИМПОРТИРОВАТЬ СВОЙ ПАКЕТ
	// "project/browser"
)

func main() {
	// 1. Инициализация
	ctx := context.Background()
	fmt.Println("🚀 Запуск CLI-интерфейса управления браузером...")

	browserSvc, err := browser.NewBrowserService(ctx, false) // false = режим с окном
	if err != nil {
		log.Fatalf("❌ Ошибка запуска: %v", err)
	}
	defer browserSvc.Close() // Если есть метод Close

	// Стартовая страница
	startURL := "https:/mail.yandex.ru"
	if err := browserSvc.Navigate(startURL); err != nil {
		log.Printf("⚠️ Ошибка навигации: %v", err)
	}

	scanner := bufio.NewScanner(os.Stdin)

	// ==========================================
	// 🔄 ГЛАВНЫЙ ЦИКЛ (REPL)
	// ==========================================
	for {
		// 1. СКАНИРОВАНИЕ (Observe)
		// Делаем это в начале каждого цикла, чтобы видеть актуальное состояние
		fmt.Println("\n👀 Сканирую страницу...")
		state, err := browserSvc.Observe()
		if err != nil {
			fmt.Printf("⚠️ Ошибка Observe: %v\n", err)
		} else {
			fmt.Println("=================================================================================")
			fmt.Printf("🌍 URL: %s | 📄 Title: %s\n", state.URL, state.Title)
			fmt.Println("---------------------------------------------------------------------------------")
			fmt.Println(state.DOMSummary)
			fmt.Println("=================================================================================")
		}

		// 2. ВВОД КОМАНДЫ
		fmt.Println("\n🎮 КОМАНДЫ: [c <id>]=Click | [t <id> <text>]=Type | [s down/up]=Scroll | [goto <url>] | [b]=Back | [k enter]=Key")
		fmt.Print("👉 Введите команду > ")

		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Пустой ввод — просто обновить DOM
		}

		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])
		args := parts[1:]

		// 3. ОБРАБОТКА КОМАНД
		var actionErr error
		startTime := time.Now()

		switch cmd {
		case "q", "quit", "exit":
			fmt.Println("👋 Завершение работы.")
			return

		case "r", "refresh":
			fmt.Println("🔄 Обновление...")
			// Просто перейдет к следующей итерации и вызовет Observe

		case "goto", "go":
			if len(args) == 0 {
				fmt.Println("❌ Укажите URL. Пример: goto google.com")
				continue
			}
			url := args[0]
			if !strings.HasPrefix(url, "http") {
				url = "https://" + url
			}
			fmt.Printf("🌐 Переход на %s...\n", url)
			actionErr = browserSvc.Navigate(url)

		case "c", "click":
			if len(args) == 0 {
				fmt.Println("❌ Укажите ID. Пример: c 57")
				continue
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println("❌ ID должен быть числом")
				continue
			}
			fmt.Printf("🖱️ Клик по ID [%d]...\n", id)
			actionErr = browserSvc.Click(id)

		case "t", "type":
			if len(args) < 2 {
				fmt.Println("❌ Формат: t <id> <текст>. Пример: t 22 привет")
				continue
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println("❌ ID должен быть числом")
				continue
			}
			text := strings.Join(args[1:], " ") // Собираем остальной текст
			fmt.Printf("⌨️ Ввод '%s' в ID [%d]...\n", text, id)
			actionErr = browserSvc.Type(id, text)

		case "s", "scroll":
			direction := "down"
			if len(args) > 0 {
				direction = args[0]
			}
			fmt.Printf("📜 Скролл %s...\n", direction)
			actionErr = browserSvc.Scroll(direction)

		case "b", "back":
			fmt.Println("⬅️ Назад...")
			actionErr = browserSvc.GoBack()

		case "k", "key":
			if len(args) == 0 {
				fmt.Println("❌ Укажите клавишу. Пример: k enter, k esc")
				continue
			}
			key := args[0]
			fmt.Printf("🎹 Нажатие клавиши: %s...\n", key)
			actionErr = browserSvc.PressKey(key)

		case "help", "h", "?":
			printHelp()
			continue // Не перерисовываем DOM, чтобы не спамить

		default:
			fmt.Println("❌ Неизвестная команда. Введите 'help' или 'h'.")
			continue
		}

		// 4. ОТЧЕТ О РЕЗУЛЬТАТЕ
		duration := time.Since(startTime)
		if actionErr != nil {
			fmt.Printf("\n❌ ОШИБКА: %v\n", actionErr)
			fmt.Println("Нажмите Enter, чтобы продолжить...")
			scanner.Scan() // Пауза, чтобы прочитать ошибку
		} else {
			fmt.Printf("\n✅ Успешно (за %v)\n", duration)
			// Небольшая пауза для визуального комфорта перед обновлением DOM
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func printHelp() {
	fmt.Println(`
📚 СПРАВКА ПО КОМАНДАМ:
---------------------------------------------
 Навигация:
   goto <url>      - Перейти по ссылке (напр. goto yandex.ru)
   b               - Назад (Back)
   r               - Обновить страницу / DOM

 Взаимодействие:
   c <id>          - Кликнуть по элементу (напр. c 57)
   t <id> <текст>  - Ввести текст (напр. t 22 айфон 15)
   s down / s up   - Скролл страницы
   k <key>         - Нажать клавишу (enter, escape, tab, backspace)

 Прочее:
   q               - Выход
   h               - Эта справка
---------------------------------------------`)
}
