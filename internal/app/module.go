package app

import (
	"log/slog"
	"os"

	bc "github.com/shandysiswandi/gobookcabin/internal/bookcabin"
)

func (a *App) initModules() {
	if a.config.GetBool("modules.book-cabin.enabled") {
		if err := bc.New(bc.Dependency{
			Config: a.config,
			Router: a.router,
		}); err != nil {
			slog.Error("failed to init module book-cabin", "error", err)
			os.Exit(1)
		}
	}
}
