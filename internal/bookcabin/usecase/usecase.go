package usecase

import (
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/cache"
	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/provider"
)

type Dependency struct {
	Providers          []provider.Provider
	Cache              *cache.Cache[*FlightsOutput]
	CacheTTL           time.Duration
	ProviderTimeout    time.Duration
	MaxProviderRetries int
}

type Usecase struct {
	providers          []provider.Provider
	cache              *cache.Cache[*FlightsOutput]
	cacheTTL           time.Duration
	providerTimeout    time.Duration
	maxProviderRetries int
}

func New(dep Dependency) *Usecase {
	return &Usecase{
		providers:          dep.Providers,
		cache:              dep.Cache,
		cacheTTL:           dep.CacheTTL,
		providerTimeout:    dep.ProviderTimeout,
		maxProviderRetries: dep.MaxProviderRetries,
	}
}
