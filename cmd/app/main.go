package main

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/app"
)

func init() {
	log.SetReportCaller(true)
	callerPrettyfier := func(f *runtime.Frame) (function string, file string) {
		if parts := strings.SplitAfter(f.File, "github.com/timohahaa"); len(parts) > 1 {
			return "", fmt.Sprintf(" %s:%d", parts[1], f.Line)
		}
		return "", fmt.Sprintf(" %s:%d", f.File, f.Line)
	}
	if term.IsTerminal(int(syscall.Stdin)) {
		log.SetFormatter(&log.TextFormatter{
			TimestampFormat:  time.DateTime,
			CallerPrettyfier: callerPrettyfier,
		})
	} else {
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat:  time.DateTime,
			CallerPrettyfier: callerPrettyfier,
		})
	}
}

//	@title			Tracker API
//	@version		0.1
//	@description	REST API трекера задач.
//
//	@servers.url			http://localhost:8080
//	@servers.description	Local development
//
// Схемы аутентификации. ВНИМАНИЕ: swag v2.0.0-rc5 умеет вывести только ОДНУ
// securityDefinitions (getSecurityDefinitionKey берёт первый ключ и остальные
// схлопываются в него), поэтому объявляем основную — сессионную cookie.
// TODO(swag): добавить вторую схему BearerToken (apiKey, in header, Authorization)
// для /mcp, когда rc5 научится нескольким схемам или после ручной пост-обработки.
//
//	@securityDefinitions.apikey	SessionCookie
//	@in							cookie
//	@name						session_id
func main() {
	var cfg config.Config
	arg.MustParse(&cfg)
	cfg.SetDefaults()

	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	a, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
