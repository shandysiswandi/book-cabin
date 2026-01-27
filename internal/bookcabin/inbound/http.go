package inbound

import (
	"context"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/usecase"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkgrouter"
)

type uc interface {
	Flights(ctx context.Context, in usecase.FlightsInput) (*usecase.FlightsOutput, error)
}

func RegisterHTTPEndpoint(r *pkgrouter.Router, uc uc) {
	end := &HTTPEndpoint{uc: uc}

	r.GET("/flights", end.Flights)
}
