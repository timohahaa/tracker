package mcp

import (
	"net/http"

	"github.com/timohahaa/tracker/internal/service"
)

// Server — MCP endpoint (/mcp, HTTP Streamable). Обёртка над service layer.
type Server struct {
	services *service.Services
}

// New создаёт MCP-обработчик. Регистрация tools и транспорт — при реализации.
func New(services *service.Services) http.Handler {
	s := &Server{services: services}
	return http.HandlerFunc(s.handle)
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
