package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/entity"
)

type BatikAirProvider struct {
	path string
	rng  *SafeRand
}

func NewBatikAirProvider(path string) *BatikAirProvider {
	return &BatikAirProvider{path: path, rng: NewSafeRand()}
}

func (b *BatikAirProvider) Name() string {
	return "Batik Air"
}

func (b *BatikAirProvider) Search(ctx context.Context, _ SearchRequest) ([]entity.Flight, error) {
	delay := time.Duration(200+b.rng.Intn(201)) * time.Millisecond
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(delay):
	}

	path := filepath.Clean(b.path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("batik air read file: %w", err)
	}

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Results []struct {
			FlightNumber      string `json:"flightNumber"`
			AirlineName       string `json:"airlineName"`
			AirlineIATA       string `json:"airlineIATA"`
			Origin            string `json:"origin"`
			Destination       string `json:"destination"`
			DepartureDateTime string `json:"departureDateTime"`
			ArrivalDateTime   string `json:"arrivalDateTime"`
			TravelTime        string `json:"travelTime"`
			NumberOfStops     int    `json:"numberOfStops"`
			Fare              struct {
				TotalPrice   int    `json:"totalPrice"`
				CurrencyCode string `json:"currencyCode"`
				Class        string `json:"class"`
			} `json:"fare"`
			SeatsAvailable int      `json:"seatsAvailable"`
			AircraftModel  string   `json:"aircraftModel"`
			BaggageInfo    string   `json:"baggageInfo"`
			Services       []string `json:"onboardServices"`
		} `json:"results"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("batik air decode: %w", err)
	}

	flights := make([]entity.Flight, 0, len(resp.Results))
	for _, f := range resp.Results {
		departAt, err := parseTimeWithLayout(f.DepartureDateTime, "2006-01-02T15:04:05-0700")
		if err != nil {
			return nil, fmt.Errorf("batik air departure time: %w", err)
		}
		arriveAt, err := parseTimeWithLayout(f.ArrivalDateTime, "2006-01-02T15:04:05-0700")
		if err != nil {
			return nil, fmt.Errorf("batik air arrival time: %w", err)
		}

		aircraft := strings.TrimSpace(f.AircraftModel)
		var aircraftPtr *string
		if aircraft != "" {
			aircraftPtr = &aircraft
		}

		carryOn, checked := splitBatikBaggage(f.BaggageInfo)

		duration := durationMinutes(departAt, arriveAt, parseDurationMinutes(f.TravelTime))
		flights = append(flights, entity.Flight{
			ID:             fmt.Sprintf("%s_%s", f.FlightNumber, b.Name()),
			Provider:       b.Name(),
			Airline:        entity.Airline{Name: f.AirlineName, Code: f.AirlineIATA},
			FlightNumber:   f.FlightNumber,
			Departure:      entity.FlightPoint{Airport: f.Origin, City: cityFromAirport(f.Origin), Time: departAt},
			Arrival:        entity.FlightPoint{Airport: f.Destination, City: cityFromAirport(f.Destination), Time: arriveAt},
			DurationMinute: duration,
			Stops:          f.NumberOfStops,
			Price:          entity.Price{Amount: f.Fare.TotalPrice, Currency: f.Fare.CurrencyCode},
			AvailableSeats: f.SeatsAvailable,
			CabinClass:     strings.ToLower(f.Fare.Class),
			Aircraft:       aircraftPtr,
			Amenities:      append([]string{}, f.Services...),
			Baggage:        entity.Baggage{CarryOn: carryOn, Checked: checked},
		})
	}

	return flights, nil
}

func parseDurationMinutes(value string) int {
	re := regexp.MustCompile(`(?i)(\d+)h\s*(\d+)m`)
	matches := re.FindStringSubmatch(value)
	if len(matches) != 3 {
		return 0
	}
	var hours, minutes int
	if _, err := fmt.Sscanf(matches[1], "%d", &hours); err != nil {
		return 0
	}
	if _, err := fmt.Sscanf(matches[2], "%d", &minutes); err != nil {
		return 0
	}
	return hours*60 + minutes
}

func splitBatikBaggage(value string) (string, string) {
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return "", ""
	}
	carryOn := strings.TrimSpace(parts[0])
	checked := ""
	if len(parts) > 1 {
		checked = strings.TrimSpace(parts[1])
	}
	return carryOn, checked
}
