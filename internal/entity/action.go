package entity

// ActionRecord — запись в истории о совершенном действии
// Это нужно для формирования промпта ("Память агента")
type ActionRecord struct {
	Reasoning string // Мысль перед действием
	Action    string // Название (click)
	Args      string // Аргументы строкой (для экономии токенов и удобства чтения LLM)
	Result    string // Результат (Success / Error)
}
