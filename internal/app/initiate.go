package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rs/cors"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkgconfig"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkgrouter"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkguid"
)

func (a *App) initConfig() {
	path := "/config/config.yaml"
	if os.Getenv("LOCAL") == "true" {
		path = "./config/config.yaml"
	}

	cfg, err := pkgconfig.NewViper(path)
	if err != nil {
		slog.Error("failed to init config", "error", err)
		os.Exit(1)
	}

	//nolint:errcheck,gosec // ignore error
	os.Setenv("TZ", cfg.GetString("app.tz"))

	a.config = cfg
}

func (a *App) initHTTPServer() {
	a.uuid = pkguid.NewUUID()
	a.router = pkgrouter.NewRouter(a.uuid)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	a.httpServer = &http.Server{
		Addr:              a.config.GetString("app.server.address.http"),
		Handler:           corsHandler.Handler(a.router),
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func (a *App) initClosers() {
	if a.closerFn == nil {
		a.closerFn = map[string]func(context.Context) error{}
	}
	a.closerFn["Config"] = func(context.Context) error {
		return a.config.Close()
	}
}
