package browser

import (
	"browser-agent/internal/entity"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func (s *BrowserService) Observe() (*entity.BrowserState, error) {
	// 1. Проверка живости вкладки (без изменений)
	if s.CurrentPage != nil {
		if _, err := s.CurrentPage.Info(); err != nil {
			fmt.Println("⚠️ Текущая вкладка мертва. Ищу живые...")
			s.CurrentPage = nil
		}
	}
	if s.CurrentPage == nil {
		pages, err := s.browser.Pages()
		if err == nil && len(pages) > 0 {
			fmt.Println("🔄 Переключился на другую открытую вкладку.")
			s.CurrentPage = pages[0]
		} else {
			fmt.Println("🆕 Все вкладки закрыты. Создаю новую...")
			page, err := s.browser.Page(proto.TargetCreateTarget{URL: "google.com"})
			if err != nil {
				return nil, fmt.Errorf("не удалось воскресить браузер: %w", err)
			}
			s.CurrentPage = page
		}
	}

	// 2. Очищаем карту
	s.ElementMap = make(map[int]*rod.Element)

	info, err := s.CurrentPage.Info()
	if err != nil {
		return nil, err
	}

	// 3. ⚡ БЫСТРОЕ ожидание — только 1-2 секунды
	tryWaitStable(s.CurrentPage, 2*time.Second)

	// 4. Выполняем JS с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := s.CurrentPage.Context(ctx).Eval(ObserveElementsScript)
	if err != nil {
		fmt.Printf("⚠️ Ошибка JS-парсинга: %v\n", err)
		return &entity.BrowserState{
			URL:        info.URL,
			Title:      info.Title,
			DOMSummary: "⚠️ Page is loading... (JS timed out)",
		}, nil
	}

	jsonString := res.Value.String()
	if jsonString == "" || jsonString == "null" {
		return &entity.BrowserState{
			URL:        info.URL,
			Title:      info.Title,
			DOMSummary: "Page is empty",
		}, nil
	}

	var elements []struct {
		ID          int    `json:"id"`
		Tag         string `json:"tag"`
		Text        string `json:"text"`
		Role        string `json:"role"`
		Interactive bool   `json:"interactive"`
	}

	if err := json.Unmarshal([]byte(jsonString), &elements); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	// 5. ⚡ СТРОИМ SUMMARY БЕЗ ЗАПРОСОВ К БРАУЗЕРУ
	var sb strings.Builder

	for _, el := range elements {
		// ❌ УБРАЛИ: s.CurrentPage.Element() — это было медленно!
		// Элементы найдём ЛЕНИВО при клике/вводе

		if el.Interactive {
			sb.WriteString(fmt.Sprintf("[%d] <%s> %s\n", el.ID, el.Tag, el.Text))
		} else {
			sb.WriteString(fmt.Sprintf("    <%s> %s\n", el.Tag, el.Text))
		}
	}

	if len(elements) >= 300 {
		sb.WriteString("\n... (truncated) ...\n")
	}

	domSummary := sb.String()
	if domSummary == "" {
		domSummary = "No elements found"
	}

	return &entity.BrowserState{
		URL:        info.URL,
		Title:      info.Title,
		DOMSummary: domSummary,
	}, nil
}

// ⚡ ЛЕНИВЫЙ поиск элемента — только когда нужен клик/ввод
func (s *BrowserService) GetElement(id int) (*rod.Element, error) {
	// Проверяем кэш
	if el, ok := s.ElementMap[id]; ok {
		return el, nil
	}

	// Ищем по data-agent-id
	selector := fmt.Sprintf("[data-agent-id='%d']", id)
	el, err := s.CurrentPage.Timeout(2 * time.Second).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element %d not found: %w", id, err)
	}

	// Кэшируем
	s.ElementMap[id] = el
	return el, nil
}

func tryWaitStable(page *rod.Page, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		page.Timeout(timeout).WaitStable(500 * time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		return
	}
}
