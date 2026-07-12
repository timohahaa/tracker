package mcp

import (
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/timohahaa/tracker/internal/errs"
)

// mcpErrorBody — тело доменной ошибки в MCP tool-результате. Форма повторяет
// HTTP-конверт ошибок (internal/handler/httpx: code/message/detail/fields),
// чтобы контракт ошибок был одинаков на обоих транспортах.
type mcpErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Detail  string            `json:"detail,omitempty"`
	Fields  []errs.FieldError `json:"fields,omitempty"`
}

// respondMCPError маппит доменную ошибку в tool-результат с IsError:true и
// JSON-описанием в текстовом content. Ошибка отдаётся как tool-ошибка (не
// протокольная), чтобы модель её видела и могла отреагировать. Не-AppError →
// internal (внутренние детали наружу не текут) — та же таксономия, что и в
// httpx.RespondError.
func respondMCPError(err error) *mcp.CallToolResult {
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		appErr = errs.Internal("internal error")
	}
	text, merr := json.Marshal(mcpErrorBody{
		Code:    string(appErr.Code),
		Message: appErr.Message,
		Detail:  appErr.Detail,
		Fields:  appErr.Fields,
	})
	if merr != nil {
		text = []byte(`{"code":"internal","message":"internal error"}`)
	}
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
	}
}
