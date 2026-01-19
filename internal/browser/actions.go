package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

func (s *BrowserService) Click(id int) error {
	el, err := s.GetElement(id)
	if err != nil {
		return fmt.Errorf("элемент ID %d не найден: %w", id, err)
	}

	pagesBefore, _ := s.browser.Pages()
	existingIDs := make(map[string]bool)
	for _, p := range pagesBefore {
		info, err := p.Info()
		if err == nil {
			existingIDs[string(info.TargetID)] = true
		}
	}

	// 2. Подсветка (с таймаутом)
	highlightCtx, highlightCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer highlightCancel()
	_, _ = el.Context(highlightCtx).Eval(HighlightClickScript)

	// 3. Контекст с таймаутом для клика
	clickCtx, clickCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer clickCancel()

	elWithTimeout := el.Context(clickCtx)

	// 4. Пытаемся кликнуть
	err = elWithTimeout.Click(proto.InputMouseButtonLeft, 1)

	// 5. Если ошибка — пробуем JS
	if err != nil {
		fmt.Printf("⚠️ Обычный клик не удался (%v), пробую JS...\n", err)
		jsErr := s.forceClickJS(el)
		if jsErr != nil {
			return fmt.Errorf("все методы клика провалились: %w", jsErr)
		}
	}

	// 6. Проверяем новую вкладку
	newPage := s.waitForNewTab(existingIDs, 3*time.Second)

	if newPage != nil {
		fmt.Printf("🔀 Новая вкладка: %s\n", safeGetURL(newPage))
		s.activatePage(newPage)
	} else {
		s.safeWaitLoad(2 * time.Second)
	}

	// 7. ⚡ ВАЖНО: Очищаем кэш после клика (DOM изменился!)
	s.ElementMap = make(map[int]*rod.Element)

	return nil
}

// forceClickJS — принудительный клик через JavaScript
func (s *BrowserService) forceClickJS(el *rod.Element) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := el.Context(ctx).Eval(`() => {
		this.click();
		this.dispatchEvent(new MouseEvent('click', {bubbles: true}));
	}`)
	return err
}

func (s *BrowserService) Type(id int, text string) error {
	// ✅ Используем GetElement() вместо прямого доступа к map
	el, err := s.GetElement(id)
	if err != nil {
		return fmt.Errorf("элемент ID %d не найден: %w", id, err)
	}

	// Подсветка
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = el.Context(ctx).Eval(HighlightTypeScript)

	// Выделяем весь текст (чтобы заменить)
	if err := el.SelectAllText(); err != nil {
		fmt.Printf("⚠️ Не удалось выделить текст: %v\n", err)
	}

	// Вводим новый текст
	if err := el.Input(text); err != nil {
		return fmt.Errorf("ошибка ввода текста: %w", err)
	}

	// ⚡ Очищаем кэш — DOM мог измениться
	s.ElementMap = make(map[int]*rod.Element)

	return nil
}

// ============================================================
// READ TEXT — чтение текста из элемента
// ============================================================
func (s *BrowserService) ReadText(id int) (string, error) {
	// ✅ Используем GetElement()
	el, err := s.GetElement(id)
	if err != nil {
		return "", fmt.Errorf("элемент ID %d не найден: %w", id, err)
	}

	// Подсветка
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = el.Context(ctx).Eval(`() => { this.style.border = "3px dashed orange" }`)

	// Чтение текста
	val, err := el.Context(ctx).Eval(`() => {
		return this.innerText || this.textContent || this.value || "";
	}`)

	if err != nil {
		return "", fmt.Errorf("JS error reading text: %w", err)
	}

	text := val.Value.String()

	// Лимит
	if len(text) > 5000 {
		text = text[:5000] + "...(truncated)"
	}

	return text, nil
}

// ============================================================
// SCROLL — прокрутка страницы
// ============================================================
func (s *BrowserService) Scroll(direction string) error {
	var script string
	if direction == "down" {
		script = ScrollDownScript
	} else {
		script = ScrollUpScript
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := s.CurrentPage.Context(ctx).Eval(script)

	time.Sleep(500 * time.Millisecond)

	// ⚡ Очищаем кэш — после скролла элементы могут измениться
	s.ElementMap = make(map[int]*rod.Element)

	return err
}

// ============================================================
// CLOSE TAB — закрытие вкладки
// ============================================================
func (s *BrowserService) CloseTab() error {
	pages, err := s.browser.Pages()
	if err != nil {
		return err
	}

	if len(pages) <= 1 {
		return fmt.Errorf("нельзя закрыть единственную вкладку, используй navigate")
	}

	// Закрываем текущую
	s.CurrentPage.Close()

	// Получаем обновленный список
	newPages, _ := s.browser.Pages()
	if len(newPages) == 0 {
		return fmt.Errorf("все вкладки закрыты")
	}

	lastPage := newPages[len(newPages)-1]
	s.activatePage(lastPage)

	// ⚡ Очищаем кэш — другая страница
	s.ElementMap = make(map[int]*rod.Element)

	fmt.Println("🔙 Вкладка закрыта, вернулись к предыдущей.")
	return nil
}

// ============================================================
// GO BACK — кнопка "Назад"
// ============================================================
func (s *BrowserService) GoBack() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.CurrentPage.Context(ctx).NavigateBack(); err != nil {
		return err
	}

	s.safeWaitLoad(3 * time.Second)

	// ⚡ Очищаем кэш — другая страница
	s.ElementMap = make(map[int]*rod.Element)

	return nil
}

// ============================================================
// PRESS KEY — нажатие клавиши
// ============================================================
func (s *BrowserService) PressKey(keyName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Ждём стабильности (с таймаутом!)
	_ = s.CurrentPage.Context(ctx).WaitStable(300 * time.Millisecond)

	var k input.Key

	switch keyName {
	case "enter":
		k = input.Enter
	case "escape":
		k = input.Escape
	case "tab":
		k = input.Tab
	case "backspace":
		k = input.Backspace
	case "arrow_down":
		k = input.ArrowDown
	case "arrow_up":
		k = input.ArrowUp
	case "space":
		k = input.Space
	default:
		return fmt.Errorf("unsupported key: %s", keyName)
	}

	err := s.CurrentPage.Keyboard.Press(k)
	if err != nil {
		return err
	}

	time.Sleep(500 * time.Millisecond)

	// ⚡ Очищаем кэш — DOM мог измениться после Enter и т.д.
	s.ElementMap = make(map[int]*rod.Element)

	return nil
}

// ============================================================
// NAVIGATE — переход на страницу
// ============================================================
func (s *BrowserService) Navigate(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := s.CurrentPage.Context(ctx).Navigate(url)
	if err != nil {
		return err
	}

	s.safeWaitLoad(5 * time.Second)

	// ⚡ Очищаем кэш — новая страница
	s.ElementMap = make(map[int]*rod.Element)

	return nil
}

// ============================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ (без изменений, но с таймаутами)
// ============================================================

func (s *BrowserService) waitForNewTab(existingIDs map[string]bool, timeout time.Duration) *rod.Page {
	deadline := time.After(timeout)
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return nil
		case <-ticker.C:
			pages, err := s.browser.Pages()
			if err != nil {
				continue
			}
			for _, p := range pages {
				info, err := p.Info()
				if err != nil {
					continue
				}
				if !existingIDs[string(info.TargetID)] {
					return p
				}
			}
		}
	}
}

func (s *BrowserService) safeWaitLoad(timeout time.Duration) {
	done := make(chan bool, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("⚠️ Паника при ожидании загрузки: %v\n", r)
			}
			done <- true
		}()

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		s.CurrentPage.Context(ctx).WaitLoad()
	}()

	select {
	case <-done:
	case <-time.After(timeout + 1*time.Second):
		fmt.Println("⚠️ Таймаут загрузки страницы, продолжаю...")
	}
}

func (s *BrowserService) activatePage(page *rod.Page) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("⚠️ Ошибка активации вкладки: %v\n", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	page.Context(ctx).Activate()
	s.CurrentPage = page

	// ⚡ Очищаем кэш — другая страница
	s.ElementMap = make(map[int]*rod.Element)

	s.safeWaitLoad(3 * time.Second)
}

func safeGetURL(page *rod.Page) string {
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	info, err := page.Context(ctx).Info()
	if err != nil {
		return "<url unavailable>"
	}
	return info.URL
}
