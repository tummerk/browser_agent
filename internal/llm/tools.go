package llm

import "github.com/openai/openai-go/v3"

func defineTools() []openai.ChatCompletionToolUnionParam {
	return []openai.ChatCompletionToolUnionParam{
		// 1. CLICK - Клик по элементу
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "click",
			Description: openai.String("Кликнуть по элементу (ссылка, кнопка, чекбокс)."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "integer",
						"description": "ID элемента из DOM (число в квадратных скобках).",
					},
				},
				"required": []string{"id"},
			},
		}),

		// 2. TYPE - Ввод текста
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "type",
			Description: openai.String("Ввести текст в поле ввода."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "integer",
						"description": "ID элемента input или textarea.",
					},
					"text": map[string]any{
						"type":        "string",
						"description": "Текст, который нужно ввести.",
					},
				},
				"required": []string{"id", "text"},
			},
		}),
		// 8. DONE - Завершение задачи
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "submit_task_result", // <--- Новое имя
			Description: openai.String("Вызови эту функцию, чтобы сдать финальный отчет и завершить работу агента."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"final_report": map[string]any{ // <--- Новое поле
						"type":        "string",
						"description": "Подробный результат выполнения задачи для пользователя.",
					},
				},
				"required": []string{"final_report"},
			},
		}),

		// 3. PRESS - Нажатие спецклавиш
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "press",
			Description: openai.String("Нажать специальную клавишу (например, Enter после ввода)."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":        "string",
						"description": "Название клавиши.",
						// Ограничиваем список, чтобы модель не придумывала свои названия
						"enum": []string{"Enter", "Backspace", "Escape", "Tab", "Delete", "ArrowDown", "ArrowUp"},
					},
				},
				"required": []string{"key"},
			},
		}),

		// 4. SCROLL - Прокрутка страницы
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "scroll",
			Description: openai.String("Прокрутить страницу, если нужный элемент не виден."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"direction": map[string]any{
						"type":        "string",
						"description": "Направление прокрутки.",
						"enum":        []string{"up", "down"},
					},
				},
				"required": []string{"direction"},
			},
		}),

		// 5. NAVIGATE - Переход по URL
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "navigate",
			Description: openai.String("Перейти на конкретный URL. Использовать только для начала работы или если ссылка не кликабельна."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "Полный URL адрес (например, https://ya.ru).",
					},
				},
				"required": []string{"url"},
			},
		}),

		// 7. MEMORIZE - Память агента
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "memorize",
			Description: openai.String("Сохранить важную информацию в память (например, содержимое письма или статус задачи)."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"info": map[string]any{
						"type":        "string",
						"description": "Факт или данные, которые нужно запомнить.",
					},
				},
				"required": []string{"info"},
			},
		}),
	}
}
