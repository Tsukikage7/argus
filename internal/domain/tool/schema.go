package tool

import "encoding/json"

// FunctionDef 是 OpenAI function calling 中的函数定义
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolDef 是 OpenAI tools 数组中的元素
type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// ToToolDefs 将 Registry 中的工具转为 OpenAI function calling 格式
func ToToolDefs(tools []Tool) []ToolDef {
	defs := make([]ToolDef, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, ToolDef{
			Type: "function",
			Function: FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}
