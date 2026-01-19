package entity

// ToolCall — намерение агента совершить действие (парсится из ответа LLM)
type ToolCall struct {
	Name      string                 // click, type, etc.
	Args      map[string]interface{} // map["id": 10, "text": "foo"]
	Reasoning string                 // "Chain of Thought" - почему он это делает
}
