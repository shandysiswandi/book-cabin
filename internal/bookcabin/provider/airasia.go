package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/entity"
)

type AirAsiaProvider struct {
	path string
	rng  *SafeRand
}

func NewAirAsiaProvider(path string) *AirAsiaProvider {
	return &AirAsiaProvider{path: path, rng: NewSafeRand()}
}

func (a *AirAsiaProvider) Name() string {
	return "AirAsia"
}

func (a *AirAsiaProvider) Search(ctx context.Context, _ SearchRequest) ([]entity.Flight, error) {
	delay := time.Duration(50+a.rng.Intn(101)) * time.Millisecond
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(delay):
	}

	if a.rng.Float64() > 0.9 {
		return nil, ErrTemporary
	}

	path := filepath.Clean(a.path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("airasia read file: %w", err)
	}

	var resp struct {
		Status  string `json:"status"`
		Flights []struct {
			FlightCode    string  `json:"flight_code"`
			Airline       string  `json:"airline"`
			FromAirport   string  `json:"from_airport"`
			ToAirport     string  `json:"to_airport"`
			DepartTime    string  `json:"depart_time"`
			ArriveTime    string  `json:"arrive_time"`
			DurationHours float64 `json:"duration_hours"`
			DirectFlight  bool    `json:"direct_flight"`
			PriceIDR      int     `json:"price_idr"`
			Seats         int     `json:"seats"`
			CabinClass    string  `json:"cabin_class"`
			BaggageNote   string  `json:"baggage_note"`
			Stops         []struct {
				Airport         string `json:"airport"`
				WaitTimeMinutes int    `json:"wait_time_minutes"`
			} `json:"stops"`
		} `json:"flights"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("airasia decode: %w", err)
	}

	flights := make([]entity.Flight, 0, len(resp.Flights))
	for _, f := range resp.Flights {
		departAt, err := time.Parse(time.RFC3339, f.DepartTime)
		if err != nil {
			return nil, fmt.Errorf("airasia departure time: %w", err)
		}
		arriveAt, err := time.Parse(time.RFC3339, f.ArriveTime)
		if err != nil {
			return nil, fmt.Errorf("airasia arrival time: %w", err)
		}

		stops := 0
		if !f.DirectFlight {
			stops = len(f.Stops)
			if stops == 0 {
				stops = 1
			}
		}

		carryOn, checked := splitAirAsiaBaggage(f.BaggageNote)

		duration := durationMinutes(departAt, arriveAt, int(math.Round(f.DurationHours*60)))
		flights = append(flights, entity.Flight{
			ID:             fmt.Sprintf("%s_%s", f.FlightCode, a.Name()),
			Provider:       a.Name(),
			Airline:        entity.Airline{Name: f.Airline, Code: strings.ToUpper(f.FlightCode[:2])},
			FlightNumber:   f.FlightCode,
			Departure:      entity.FlightPoint{Airport: f.FromAirport, City: cityFromAirport(f.FromAirport), Time: departAt},
			Arrival:        entity.FlightPoint{Airport: f.ToAirport, City: cityFromAirport(f.ToAirport), Time: arriveAt},
			DurationMinute: duration,
			Stops:          stops,
			Price:          entity.Price{Amount: f.PriceIDR, Currency: "IDR"},
			AvailableSeats: f.Seats,
			CabinClass:     strings.ToLower(f.CabinClass),
			Amenities:      []string{},
			Baggage:        entity.Baggage{CarryOn: carryOn, Checked: checked},
		})
	}

	return flights, nil
}

func splitAirAsiaBaggage(note string) (string, string) {
	if note == "" {
		return "", ""
	}
	parts := strings.Split(note, ",")
	if len(parts) == 0 {
		return "", ""
	}
	carryOn := strings.TrimSpace(parts[0])
	checked := ""
	if len(parts) > 1 {
		checked = strings.TrimSpace(parts[1])
		checked = strings.TrimPrefix(checked, "checked bags ")
		checked = strings.TrimPrefix(checked, "checked ")
	}
	if checked == "" {
		checked = "Additional fee"
	}
	return carryOn, checked
}
