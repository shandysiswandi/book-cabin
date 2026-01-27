package provider

import (
	"context"
	"errors"
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/entity"
)

var ErrTemporary = errors.New("temporary provider error")

type SearchRequest struct {
	Origin        string
	Destination   string
	DepartureDate time.Time
	ReturnDate    *time.Time
	Passengers    int
	CabinClass    string
}

type Provider interface {
	Name() string
	Search(ctx context.Context, req SearchRequest) ([]entity.Flight, error)
}
