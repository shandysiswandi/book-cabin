package inbound

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/entity"
	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/usecase"
	"github.com/shandysiswandi/gobookcabin/internal/pkg/pkgerror"
)

func parseFlightsInput(r *http.Request) (usecase.FlightsInput, error) {
	q := r.URL.Query()

	origin := strings.TrimSpace(q.Get("origin"))
	destination := strings.TrimSpace(q.Get("destination"))
	if origin == "" || destination == "" {
		return usecase.FlightsInput{}, pkgerror.NewBusiness("origin and destination are required", pkgerror.CodeInvalidInput)
	}

	departureDateStr := strings.TrimSpace(firstNotEmpty(q.Get("departureDate"), q.Get("departure_date")))
	if departureDateStr == "" {
		return usecase.FlightsInput{}, pkgerror.NewBusiness("departureDate is required", pkgerror.CodeInvalidInput)
	}
	departureDate, err := time.ParseInLocation("2006-01-02", departureDateStr, time.Local)
	if err != nil {
		return usecase.FlightsInput{}, pkgerror.NewBusiness("invalid departureDate", pkgerror.CodeInvalidInput)
	}

	var returnDate *time.Time
	returnDateStr := strings.TrimSpace(firstNotEmpty(q.Get("returnDate"), q.Get("return_date")))
	if returnDateStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", returnDateStr, time.Local)
		if err != nil {
			return usecase.FlightsInput{}, pkgerror.NewBusiness("invalid returnDate", pkgerror.CodeInvalidInput)
		}
		returnDate = &parsed
	}

	passengers := 1
	if value := strings.TrimSpace(q.Get("passengers")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return usecase.FlightsInput{}, pkgerror.NewBusiness("invalid passengers", pkgerror.CodeInvalidInput)
		}
		passengers = parsed
	}

	cabinClass := strings.TrimSpace(firstNotEmpty(q.Get("cabinClass"), q.Get("cabin_class")))
	if cabinClass == "" {
		cabinClass = "economy"
	}

	filters, err := parseFlightFilters(q, departureDate)
	if err != nil {
		return usecase.FlightsInput{}, err
	}

	sortOpt := usecase.SortOption{
		Field: strings.TrimSpace(q.Get("sort")),
		Order: strings.TrimSpace(q.Get("order")),
	}

	return usecase.FlightsInput{
		Origin:        origin,
		Destination:   destination,
		DepartureDate: departureDate,
		ReturnDate:    returnDate,
		Passengers:    passengers,
		CabinClass:    strings.ToLower(cabinClass),
		Filters:       filters,
		Sort:          sortOpt,
	}, nil
}

func parseFlightFilters(q url.Values, departureDate time.Time) (usecase.FlightFilters, error) {
	filters := usecase.FlightFilters{}
	if err := parseIntFilter(q, "min_price", "minPrice", "invalid min_price", &filters.MinPrice); err != nil {
		return filters, err
	}
	if err := parseIntFilter(q, "max_price", "maxPrice", "invalid max_price", &filters.MaxPrice); err != nil {
		return filters, err
	}
	if err := parseIntFilter(q, "stops", "stop_count", "invalid stops", &filters.Stops); err != nil {
		return filters, err
	}
	if err := parseIntFilter(q, "max_stops", "maxStops", "invalid max_stops", &filters.MaxStops); err != nil {
		return filters, err
	}
	if err := parseIntFilter(q, "min_duration", "minDuration", "invalid min_duration", &filters.MinDuration); err != nil {
		return filters, err
	}
	if err := parseIntFilter(q, "max_duration", "maxDuration", "invalid max_duration", &filters.MaxDuration); err != nil {
		return filters, err
	}
	filters.Airlines = parseListFilter(q, "airlines", "airline")

	departAfter, err := parseTimeFilter(q, "depart_after", "departAfter", departureDate)
	if err != nil {
		return filters, err
	}
	filters.DepartAfter = departAfter
	departBefore, err := parseTimeFilter(q, "depart_before", "departBefore", departureDate)
	if err != nil {
		return filters, err
	}
	filters.DepartBefore = departBefore
	arriveAfter, err := parseTimeFilter(q, "arrive_after", "arriveAfter", departureDate)
	if err != nil {
		return filters, err
	}
	filters.ArriveAfter = arriveAfter
	arriveBefore, err := parseTimeFilter(q, "arrive_before", "arriveBefore", departureDate)
	if err != nil {
		return filters, err
	}
	filters.ArriveBefore = arriveBefore

	return filters, nil
}

func parseIntFilter(q url.Values, key, altKey, errMsg string, target **int) error {
	value := strings.TrimSpace(firstNotEmpty(q.Get(key), q.Get(altKey)))
	if value == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return pkgerror.NewBusiness(errMsg, pkgerror.CodeInvalidInput)
	}
	*target = &parsed
	return nil
}

func parseListFilter(q url.Values, key, altKey string) []string {
	value := strings.TrimSpace(firstNotEmpty(q.Get(key), q.Get(altKey)))
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}

func parseTimeFilter(q url.Values, key, altKey string, date time.Time) (*time.Time, error) {
	value := strings.TrimSpace(firstNotEmpty(q.Get(key), q.Get(altKey)))
	if value == "" {
		return nil, nil
	}
	if len(value) == 5 && strings.Contains(value, ":") {
		parsed, err := time.ParseInLocation("15:04", value, time.Local)
		if err != nil {
			return nil, pkgerror.NewBusiness("invalid time filter", pkgerror.CodeInvalidInput)
		}
		combined := time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.Local)
		return &combined, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, pkgerror.NewBusiness("invalid time filter", pkgerror.CodeInvalidInput)
	}
	return &parsed, nil
}

func firstNotEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func mapFlightPoint(point entity.FlightPoint) FlightPoint {
	return FlightPoint{
		Airport:   point.Airport,
		City:      point.City,
		Datetime:  point.Time.Format(time.RFC3339),
		Timestamp: point.Time.Unix(),
	}
}

func formatDuration(minutes int) string {
	if minutes <= 0 {
		return ""
	}
	hours := minutes / 60
	mins := minutes % 60
	if hours == 0 {
		return fmt.Sprintf("%dm", mins)
	}
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func formatIDR(amount int) string {
	if amount == 0 {
		return "Rp. 0"
	}
	negative := amount < 0
	if negative {
		amount = -amount
	}
	value := strconv.Itoa(amount)
	for i := len(value) - 3; i > 0; i -= 3 {
		value = value[:i] + "." + value[i:]
	}
	if negative {
		return "-Rp. " + value
	}
	return "Rp. " + value
}
