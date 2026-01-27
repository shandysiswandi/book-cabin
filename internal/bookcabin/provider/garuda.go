package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/entity"
)

type GarudaIndonesiaProvider struct {
	path string
	rng  *SafeRand
}

func NewGarudaIndonesiaProvider(path string) *GarudaIndonesiaProvider {
	return &GarudaIndonesiaProvider{path: path, rng: NewSafeRand()}
}

func (g *GarudaIndonesiaProvider) Name() string {
	return "Garuda Indonesia"
}

func (g *GarudaIndonesiaProvider) Search(ctx context.Context, _ SearchRequest) ([]entity.Flight, error) {
	delay := time.Duration(50+g.rng.Intn(51)) * time.Millisecond
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(delay):
	}

	path := filepath.Clean(g.path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("garuda read file: %w", err)
	}

	var resp struct {
		Status  string `json:"status"`
		Flights []struct {
			FlightID    string `json:"flight_id"`
			Airline     string `json:"airline"`
			AirlineCode string `json:"airline_code"`
			Departure   struct {
				Airport string `json:"airport"`
				City    string `json:"city"`
				Time    string `json:"time"`
			} `json:"departure"`
			Arrival struct {
				Airport string `json:"airport"`
				City    string `json:"city"`
				Time    string `json:"time"`
			} `json:"arrival"`
			DurationMinutes int    `json:"duration_minutes"`
			Stops           int    `json:"stops"`
			Aircraft        string `json:"aircraft"`
			Price           struct {
				Amount   int    `json:"amount"`
				Currency string `json:"currency"`
			} `json:"price"`
			AvailableSeats int      `json:"available_seats"`
			FareClass      string   `json:"fare_class"`
			Amenities      []string `json:"amenities"`
			Baggage        struct {
				CarryOn int `json:"carry_on"`
				Checked int `json:"checked"`
			} `json:"baggage"`
			Segments []struct {
				FlightNumber string `json:"flight_number"`
				Departure    struct {
					Airport string `json:"airport"`
					Time    string `json:"time"`
				} `json:"departure"`
				Arrival struct {
					Airport string `json:"airport"`
					Time    string `json:"time"`
				} `json:"arrival"`
				DurationMinutes int `json:"duration_minutes"`
				LayoverMinutes  int `json:"layover_minutes"`
			} `json:"segments"`
		} `json:"flights"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("garuda decode: %w", err)
	}

	flights := make([]entity.Flight, 0, len(resp.Flights))
	for _, f := range resp.Flights {
		departAt, err := time.Parse(time.RFC3339, f.Departure.Time)
		if err != nil {
			return nil, fmt.Errorf("garuda departure time: %w", err)
		}
		arriveAt, err := time.Parse(time.RFC3339, f.Arrival.Time)
		if err != nil {
			return nil, fmt.Errorf("garuda arrival time: %w", err)
		}

		aircraft := strings.TrimSpace(f.Aircraft)
		var aircraftPtr *string
		if aircraft != "" {
			aircraftPtr = &aircraft
		}

		baggage := entity.Baggage{
			CarryOn: fmt.Sprintf("%d piece", f.Baggage.CarryOn),
			Checked: fmt.Sprintf("%d piece", f.Baggage.Checked),
		}
		if f.Baggage.CarryOn > 1 {
			baggage.CarryOn = fmt.Sprintf("%d pieces", f.Baggage.CarryOn)
		}
		if f.Baggage.Checked > 1 {
			baggage.Checked = fmt.Sprintf("%d pieces", f.Baggage.Checked)
		}

		stops := f.Stops
		if stops == 0 && len(f.Segments) > 1 {
			stops = len(f.Segments) - 1
		}

		duration := durationMinutes(departAt, arriveAt, f.DurationMinutes)
		flights = append(flights, entity.Flight{
			ID:             fmt.Sprintf("%s_%s", f.FlightID, g.Name()),
			Provider:       g.Name(),
			Airline:        entity.Airline{Name: f.Airline, Code: f.AirlineCode},
			FlightNumber:   f.FlightID,
			Departure:      entity.FlightPoint{Airport: f.Departure.Airport, City: f.Departure.City, Time: departAt},
			Arrival:        entity.FlightPoint{Airport: f.Arrival.Airport, City: f.Arrival.City, Time: arriveAt},
			DurationMinute: duration,
			Stops:          stops,
			Price:          entity.Price{Amount: f.Price.Amount, Currency: f.Price.Currency},
			AvailableSeats: f.AvailableSeats,
			CabinClass:     strings.ToLower(f.FareClass),
			Aircraft:       aircraftPtr,
			Amenities:      append([]string{}, f.Amenities...),
			Baggage:        baggage,
		})
	}

	return flights, nil
}
