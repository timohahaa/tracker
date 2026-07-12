package mcp

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/timohahaa/tracker/internal/service"
)

// Server — MCP endpoint (/mcp, HTTP Streamable). Тонкая обёртка над service
// layer: tools лишь мапят аргументы и зовут сервисы, бизнес-логика/транзакции
// остаются внутри service.
type Server struct {
	services *service.Services
}

// New создаёт MCP-обработчик. Регистрирует tools на общем mcp.Server и отдаёт
// его через Streamable HTTP в stateless-режиме: каждый POST подключается с
// контекстом запроса, поэтому actor/workspace, положенные TokenAuth в контекст,
// доходят до tool-хендлеров теми же хелперами, что и в REST.
func New(services *service.Services) http.Handler {
	s := &Server{services: services}

	srv := mcp.NewServer(&mcp.Implementation{Name: "tracker", Version: "1.0.0"}, nil)
	s.register(srv)

	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
}
