package usecase

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/entity"
)

func buildCacheKey(in FlightsInput) string {
	return fmt.Sprintf(
		"%s|%s|%s|%s|%d|%s|%s|%s|%s",
		strings.ToUpper(in.Origin),
		strings.ToUpper(in.Destination),
		in.DepartureDate.Format("2006-01-02"),
		formatOptionalDate(in.ReturnDate),
		in.Passengers,
		strings.ToLower(in.CabinClass),
		formatFilters(in.Filters),
		strings.ToLower(in.Sort.Field),
		strings.ToLower(in.Sort.Order),
	)
}

func formatOptionalDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}

func formatFilters(filters FlightFilters) string {
	parts := []string{
		formatOptionalInt(filters.MinPrice),
		formatOptionalInt(filters.MaxPrice),
		formatOptionalInt(filters.Stops),
		formatOptionalInt(filters.MaxStops),
		formatOptionalInt(filters.MinDuration),
		formatOptionalInt(filters.MaxDuration),
		formatOptionalTime(filters.DepartAfter),
		formatOptionalTime(filters.DepartBefore),
		formatOptionalTime(filters.ArriveAfter),
		formatOptionalTime(filters.ArriveBefore),
		formatAirlines(filters.Airlines),
	}
	return strings.Join(parts, ",")
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func formatAirlines(values []string) string {
	if len(values) == 0 {
		return ""
	}
	clean := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	sort.Strings(clean)
	return strings.Join(clean, "|")
}

func CloneFlightsOutput(value *FlightsOutput) *FlightsOutput {
	if value == nil {
		return nil
	}
	clone := &FlightsOutput{
		SearchCriteria: value.SearchCriteria,
		Metadata:       value.Metadata,
		Flights:        make([]entity.Flight, len(value.Flights)),
		ReturnFlights:  make([]entity.Flight, len(value.ReturnFlights)),
	}
	copy(clone.Flights, value.Flights)
	copy(clone.ReturnFlights, value.ReturnFlights)
	return clone
}
