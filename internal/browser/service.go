package browser

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

// BrowserService управляет браузером и хранит карту элементов для Агента
type BrowserService struct {
	browser     *rod.Browser
	CurrentPage *rod.Page            // Текущая активная вкладка
	ElementMap  map[int]*rod.Element // Карта ID -> Элемент (для кликов)
}

// NewBrowserService создает браузер.
func NewBrowserService(ctx context.Context, headless bool) (*BrowserService, error) {
	// 1. Настройка лаунчера
	launch := launcher.New().
		Leakless(true).
		Headless(headless).
		UserDataDir("user_data")

	controlURL, err := launch.Launch()
	if err != nil {
		return nil, fmt.Errorf("не удалось запустить браузер: %w", err)
	}

	// 2. Подключение
	browser := rod.New().ControlURL(controlURL).Context(ctx)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("не удалось подключиться: %w", err)
	}

	// 3. Создание STEALTH страницы
	page, err := stealth.Page(browser)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания stealth страницы: %w", err)
	}
	scale := 1.0

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  1920,
		Height: 1080,
		Scale:  &scale,
		Mobile: false,
	}); err != nil {
		// Не критично, можно логировать
		fmt.Printf("Warning: failed to set viewport: %v\n", err)
	}

	// Таймаут поиска элементов
	page.Timeout(10 * time.Second)

	return &BrowserService{
		browser:     browser,
		CurrentPage: page,
		ElementMap:  make(map[int]*rod.Element),
	}, nil
}

func (s *BrowserService) GetCurrentPageInfo() (string, string) {
	if s.CurrentPage == nil {
		return "", ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	info, err := s.CurrentPage.Context(ctx).Info()
	if err != nil {
		return "", ""
	}

	return info.URL, string(info.TargetID)
}

// Структура для парсинга данных из
type domElement struct {
	ID          int    `json:"id"`
	Tag         string `json:"tag"`
	Text        string `json:"text"`
	Type        string `json:"type"`
	Interactive bool   `json:"interactive"`
	State       string `json:"state"`
}

func (s *BrowserService) Close() {
	if s.browser != nil {
		err := s.browser.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
}
