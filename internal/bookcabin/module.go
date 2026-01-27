package bookcabin

import (
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/cache"
	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/inbound"
	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/provider"
	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/usecase"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkgconfig"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkgrouter"
)

type Dependency struct {
	Config pkgconfig.Config
	Router *pkgrouter.Router
}

func New(dep Dependency) error {
	providers := []provider.Provider{
		provider.NewGarudaIndonesiaProvider("mocks/garuda_indonesia_search_response.json"),
		provider.NewLionAirProvider("mocks/lion_air_search_response.json"),
		provider.NewBatikAirProvider("mocks/batik_air_search_response.json"),
		provider.NewAirAsiaProvider("mocks/airasia_search_response.json"),
	}

	rateLimit := 100 * time.Millisecond
	if rateLimitMs := dep.Config.GetInt("modules.book-cabin.provider.rate_limit_ms"); rateLimitMs > 0 {
		rateLimit = time.Duration(rateLimitMs) * time.Millisecond
	}
	if rateLimit > 0 {
		for i := range providers {
			providers[i] = provider.NewRateLimitedProvider(providers[i], rateLimit)
		}
	}

	cacheTTL := 60 * time.Second
	if ttlSeconds := dep.Config.GetInt("modules.book-cabin.cache.ttl_seconds"); ttlSeconds > 0 {
		cacheTTL = time.Duration(ttlSeconds) * time.Second
	}

	cacheStore := cache.New(usecase.CloneFlightsOutput)

	uc := usecase.New(usecase.Dependency{
		Providers:          providers,
		Cache:              cacheStore,
		CacheTTL:           cacheTTL,
		ProviderTimeout:    1 * time.Second,
		MaxProviderRetries: 2,
	})

	inbound.RegisterHTTPEndpoint(dep.Router, uc)

	return nil
}
