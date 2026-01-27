package app

import (
	"context"
	"net/http"

	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkgconfig"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkglog"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkgrouter"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkguid"
)

type App struct {
	config     pkgconfig.Config
	uuid       pkguid.StringID
	router     *pkgrouter.Router
	httpServer *http.Server
	closerFn   map[string]func(context.Context) error
}

func New() *App {
	app := &App{}
	pkglog.InitLogging()
	app.initConfig()
	app.initHTTPServer()
	app.initModules()
	app.initClosers()
	return app
}
