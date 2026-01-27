package usecase

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/entity"
	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/provider"
)

type FlightsInput struct {
	Origin        string
	Destination   string
	DepartureDate time.Time
	ReturnDate    *time.Time
	Passengers    int
	CabinClass    string
	Filters       FlightFilters
	Sort          SortOption
}

type FlightFilters struct {
	MinPrice     *int
	MaxPrice     *int
	Stops        *int
	MaxStops     *int
	MinDuration  *int
	MaxDuration  *int
	Airlines     []string
	DepartAfter  *time.Time
	DepartBefore *time.Time
	ArriveAfter  *time.Time
	ArriveBefore *time.Time
}

type SortOption struct {
	Field string
	Order string
}

type FlightsOutput struct {
	SearchCriteria SearchCriteria
	Metadata       SearchMetadata
	Flights        []entity.Flight
	ReturnFlights  []entity.Flight
}

type SearchCriteria struct {
	Origin        string
	Destination   string
	DepartureDate string
	ReturnDate    *string
	Passengers    int
	CabinClass    string
}

type SearchMetadata struct {
	TotalResults       int
	ProvidersQueried   int
	ProvidersSucceeded int
	ProvidersFailed    int
	SearchTimeMs       int64
	CacheHit           bool
	FailedProviders    []string
}

var errProviderFailed = errors.New("provider search failed")

func (u *Usecase) Flights(ctx context.Context, in FlightsInput) (*FlightsOutput, error) {
	start := time.Now()
	cacheKey := buildCacheKey(in)
	if cached, ok := u.cache.Get(cacheKey); ok {
		cached.Metadata.CacheHit = true
		cached.Metadata.SearchTimeMs = time.Since(start).Milliseconds()
		return cached, nil
	}

	outboundReq := provider.SearchRequest{
		Origin:        in.Origin,
		Destination:   in.Destination,
		DepartureDate: in.DepartureDate,
		Passengers:    in.Passengers,
		CabinClass:    in.CabinClass,
	}
	outboundFlights, outboundStats := u.collectFlights(ctx, outboundReq, in, in.Origin, in.Destination, in.DepartureDate, in.Filters)
	applyBestValueScore(outboundFlights)
	sortFlights(outboundFlights, in.Sort)

	returnFlights := []entity.Flight{}
	returnStats := providerStats{}
	if in.ReturnDate != nil {
		returnFilters := shiftFiltersDate(in.Filters, *in.ReturnDate)
		returnReq := provider.SearchRequest{
			Origin:        in.Destination,
			Destination:   in.Origin,
			DepartureDate: *in.ReturnDate,
			Passengers:    in.Passengers,
			CabinClass:    in.CabinClass,
		}
		returnFlights, returnStats = u.collectFlights(ctx, returnReq, in, in.Destination, in.Origin, *in.ReturnDate, returnFilters)
		applyBestValueScore(returnFlights)
		sortFlights(returnFlights, in.Sort)
	}

	providersSucceeded, failedProviders := mergeProviderStats(u.providers, outboundStats, returnStats)

	searchCriteria := SearchCriteria{
		Origin:        in.Origin,
		Destination:   in.Destination,
		DepartureDate: in.DepartureDate.Format("2006-01-02"),
		Passengers:    in.Passengers,
		CabinClass:    in.CabinClass,
	}
	if in.ReturnDate != nil {
		value := in.ReturnDate.Format("2006-01-02")
		searchCriteria.ReturnDate = &value
	}

	output := &FlightsOutput{
		SearchCriteria: searchCriteria,
		Metadata: SearchMetadata{
			TotalResults:       len(outboundFlights) + len(returnFlights),
			ProvidersQueried:   len(u.providers),
			ProvidersSucceeded: providersSucceeded,
			ProvidersFailed:    len(u.providers) - providersSucceeded,
			SearchTimeMs:       time.Since(start).Milliseconds(),
			CacheHit:           false,
			FailedProviders:    failedProviders,
		},
		Flights:       outboundFlights,
		ReturnFlights: returnFlights,
	}

	u.cache.Set(cacheKey, output, u.cacheTTL)

	return output, nil
}

type providerResult struct {
	name    string
	flights []entity.Flight
	err     error
}

type providerStats struct {
	success map[string]bool
	failed  map[string]bool
}

func (u *Usecase) searchProviders(ctx context.Context, req provider.SearchRequest) []providerResult {
	results := make([]providerResult, 0, len(u.providers))
	resCh := make(chan providerResult, len(u.providers))

	for _, p := range u.providers {
		providerItem := p
		go func() {
			providerCtx, cancel := context.WithTimeout(ctx, u.providerTimeout)
			defer cancel()
			flights, err := u.searchWithRetry(providerCtx, providerItem, req)
			resCh <- providerResult{name: providerItem.Name(), flights: flights, err: err}
		}()
	}

	for i := 0; i < len(u.providers); i++ {
		results = append(results, <-resCh)
	}

	return results
}

func (u *Usecase) collectFlights(
	ctx context.Context,
	req provider.SearchRequest,
	in FlightsInput,
	origin string,
	destination string,
	date time.Time,
	filters FlightFilters,
) ([]entity.Flight, providerStats) {
	results := u.searchProviders(ctx, req)
	flights := make([]entity.Flight, 0)
	stats := providerStats{success: map[string]bool{}, failed: map[string]bool{}}
	for _, res := range results {
		if res.err != nil {
			stats.failed[res.name] = true
			continue
		}
		stats.success[res.name] = true
		flights = append(flights, res.flights...)
	}
	flights = normalizeDurations(flights)
	criteriaDate := date.Format("2006-01-02")
	filtered := filterFlights(flights, origin, destination, in.CabinClass, filters, criteriaDate)
	compared := compareAndDedupFlights(filtered)
	return compared, stats
}

func (u *Usecase) searchWithRetry(ctx context.Context, p provider.Provider, req provider.SearchRequest) ([]entity.Flight, error) {
	backoff := 80 * time.Millisecond
	for attempt := 0; attempt <= u.maxProviderRetries; attempt++ {
		flights, err := p.Search(ctx, req)
		if err == nil {
			return flights, nil
		}
		if !errors.Is(err, provider.ErrTemporary) {
			return nil, err
		}
		if attempt == u.maxProviderRetries {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	return nil, errProviderFailed
}

func filterFlights(flights []entity.Flight, origin, destination, cabinClass string, filters FlightFilters, criteriaDate string) []entity.Flight {
	filtered := make([]entity.Flight, 0, len(flights))
	airlineFilter := normalizeSet(filters.Airlines)

	for _, flight := range flights {
		if !matchFlightCriteria(flight, origin, destination, cabinClass, criteriaDate) {
			continue
		}
		if !matchFilter(flight, filters, airlineFilter) {
			continue
		}
		filtered = append(filtered, flight)
	}

	return filtered
}

func matchFlightCriteria(f entity.Flight, origin, destination, cabinClass, criteriaDate string) bool {
	if !strings.EqualFold(f.Departure.Airport, origin) {
		return false
	}
	if !strings.EqualFold(f.Arrival.Airport, destination) {
		return false
	}
	if f.Departure.Time.IsZero() || f.Arrival.Time.IsZero() {
		return false
	}
	if !f.Arrival.Time.After(f.Departure.Time) {
		return false
	}
	if f.Departure.Time.Format("2006-01-02") != criteriaDate {
		return false
	}
	if cabinClass != "" && !strings.EqualFold(f.CabinClass, cabinClass) {
		return false
	}
	return true
}

func shiftFiltersDate(filters FlightFilters, date time.Time) FlightFilters {
	clone := filters
	clone.DepartAfter = shiftTimeDate(filters.DepartAfter, date)
	clone.DepartBefore = shiftTimeDate(filters.DepartBefore, date)
	clone.ArriveAfter = shiftTimeDate(filters.ArriveAfter, date)
	clone.ArriveBefore = shiftTimeDate(filters.ArriveBefore, date)
	return clone
}

func shiftTimeDate(value *time.Time, date time.Time) *time.Time {
	if value == nil {
		return nil
	}
	shifted := time.Date(date.Year(), date.Month(), date.Day(), value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), value.Location())
	return &shifted
}

func mergeProviderStats(providers []provider.Provider, outbound providerStats, inbound providerStats) (int, []string) {
	succeeded := 0
	failedProviders := make([]string, 0)
	for _, p := range providers {
		name := p.Name()
		if outbound.success[name] || inbound.success[name] {
			succeeded++
			continue
		}
		failedProviders = append(failedProviders, name)
	}
	return succeeded, failedProviders
}

func matchFilter(f entity.Flight, filters FlightFilters, airlineFilter map[string]struct{}) bool {
	if !matchPriceFilter(f, filters) {
		return false
	}
	if !matchStopsFilter(f, filters) {
		return false
	}
	if !matchDurationFilter(f, filters) {
		return false
	}
	if !matchAirlineFilter(f, airlineFilter) {
		return false
	}
	return matchTimeFilter(f, filters)
}

func matchPriceFilter(f entity.Flight, filters FlightFilters) bool {
	if filters.MinPrice != nil && f.Price.Amount < *filters.MinPrice {
		return false
	}
	if filters.MaxPrice != nil && f.Price.Amount > *filters.MaxPrice {
		return false
	}
	return true
}

func matchStopsFilter(f entity.Flight, filters FlightFilters) bool {
	if filters.Stops != nil && f.Stops != *filters.Stops {
		return false
	}
	if filters.MaxStops != nil && f.Stops > *filters.MaxStops {
		return false
	}
	return true
}

func matchDurationFilter(f entity.Flight, filters FlightFilters) bool {
	if filters.MinDuration != nil && f.DurationMinute < *filters.MinDuration {
		return false
	}
	if filters.MaxDuration != nil && f.DurationMinute > *filters.MaxDuration {
		return false
	}
	return true
}

func matchAirlineFilter(f entity.Flight, airlineFilter map[string]struct{}) bool {
	if len(airlineFilter) == 0 {
		return true
	}
	if _, ok := airlineFilter[strings.ToLower(f.Airline.Name)]; ok {
		return true
	}
	if _, ok := airlineFilter[strings.ToLower(f.Airline.Code)]; ok {
		return true
	}
	return false
}

func matchTimeFilter(f entity.Flight, filters FlightFilters) bool {
	if filters.DepartAfter != nil && f.Departure.Time.Before(*filters.DepartAfter) {
		return false
	}
	if filters.DepartBefore != nil && f.Departure.Time.After(*filters.DepartBefore) {
		return false
	}
	if filters.ArriveAfter != nil && f.Arrival.Time.Before(*filters.ArriveAfter) {
		return false
	}
	if filters.ArriveBefore != nil && f.Arrival.Time.After(*filters.ArriveBefore) {
		return false
	}
	return true
}

func applyBestValueScore(flights []entity.Flight) {
	if len(flights) == 0 {
		return
	}
	minPrice, maxPrice := flights[0].Price.Amount, flights[0].Price.Amount
	minDuration, maxDuration := flights[0].DurationMinute, flights[0].DurationMinute
	for _, f := range flights[1:] {
		if f.Price.Amount < minPrice {
			minPrice = f.Price.Amount
		}
		if f.Price.Amount > maxPrice {
			maxPrice = f.Price.Amount
		}
		if f.DurationMinute < minDuration {
			minDuration = f.DurationMinute
		}
		if f.DurationMinute > maxDuration {
			maxDuration = f.DurationMinute
		}
	}

	priceRange := float64(maxPrice - minPrice)
	durationRange := float64(maxDuration - minDuration)
	if priceRange == 0 {
		priceRange = 1
	}
	if durationRange == 0 {
		durationRange = 1
	}

	for i := range flights {
		priceScore := float64(flights[i].Price.Amount-minPrice) / priceRange
		durationScore := float64(flights[i].DurationMinute-minDuration) / durationRange
		flights[i].BestValueScore = priceScore*0.6 + durationScore*0.4
	}
}

func sortFlights(flights []entity.Flight, sortOpt SortOption) {
	field := strings.ToLower(sortOpt.Field)
	order := strings.ToLower(sortOpt.Order)
	if field == "" {
		field = "best_value"
	}
	if order == "" {
		order = "asc"
	}

	less := func(i, j int) bool {
		switch field {
		case "price":
			return flights[i].Price.Amount < flights[j].Price.Amount
		case "duration":
			return flights[i].DurationMinute < flights[j].DurationMinute
		case "departure":
			return flights[i].Departure.Time.Before(flights[j].Departure.Time)
		case "arrival":
			return flights[i].Arrival.Time.Before(flights[j].Arrival.Time)
		case "best_value":
			return flights[i].BestValueScore < flights[j].BestValueScore
		default:
			return flights[i].BestValueScore < flights[j].BestValueScore
		}
	}

	if order == "desc" {
		sort.SliceStable(flights, func(i, j int) bool { return !less(i, j) })
		return
	}

	sort.SliceStable(flights, less)
}

func normalizeSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		value := strings.ToLower(strings.TrimSpace(v))
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return set
}

func normalizeDurations(flights []entity.Flight) []entity.Flight {
	for i := range flights {
		if flights[i].Departure.Time.IsZero() || flights[i].Arrival.Time.IsZero() {
			continue
		}
		if flights[i].Arrival.Time.After(flights[i].Departure.Time) {
			duration := int(flights[i].Arrival.Time.Sub(flights[i].Departure.Time).Minutes())
			if duration > 0 {
				flights[i].DurationMinute = duration
			}
		}
	}
	return flights
}

func compareAndDedupFlights(flights []entity.Flight) []entity.Flight {
	if len(flights) == 0 {
		return flights
	}
	bestByKey := make(map[string]entity.Flight, len(flights))
	for _, flight := range flights {
		key := flightKey(flight)
		current, ok := bestByKey[key]
		if !ok || flight.Price.Amount < current.Price.Amount {
			bestByKey[key] = flight
			continue
		}
		if flight.Price.Amount == current.Price.Amount && flight.Provider < current.Provider {
			bestByKey[key] = flight
		}
	}

	unique := make([]entity.Flight, 0, len(bestByKey))
	for _, flight := range bestByKey {
		unique = append(unique, flight)
	}
	return unique
}

func flightKey(f entity.Flight) string {
	return strings.ToLower(strings.Join([]string{
		f.Airline.Code,
		f.FlightNumber,
		f.Departure.Airport,
		f.Arrival.Airport,
		f.Departure.Time.Format(time.RFC3339),
		f.Arrival.Time.Format(time.RFC3339),
	}, "|"))
}
