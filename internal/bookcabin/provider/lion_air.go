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

type LionAirProvider struct {
	path string
	rng  *SafeRand
}

func NewLionAirProvider(path string) *LionAirProvider {
	return &LionAirProvider{path: path, rng: NewSafeRand()}
}

func (l *LionAirProvider) Name() string {
	return "Lion Air"
}

func (l *LionAirProvider) Search(ctx context.Context, _ SearchRequest) ([]entity.Flight, error) {
	delay := time.Duration(100+l.rng.Intn(101)) * time.Millisecond
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(delay):
	}

	path := filepath.Clean(l.path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lion air read file: %w", err)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			AvailableFlights []struct {
				ID      string `json:"id"`
				Carrier struct {
					Name string `json:"name"`
					IATA string `json:"iata"`
				} `json:"carrier"`
				Route struct {
					From struct {
						Code string `json:"code"`
						City string `json:"city"`
					} `json:"from"`
					To struct {
						Code string `json:"code"`
						City string `json:"city"`
					} `json:"to"`
				} `json:"route"`
				Schedule struct {
					Departure         string `json:"departure"`
					DepartureTimezone string `json:"departure_timezone"`
					Arrival           string `json:"arrival"`
					ArrivalTimezone   string `json:"arrival_timezone"`
				} `json:"schedule"`
				FlightTime int  `json:"flight_time"`
				IsDirect   bool `json:"is_direct"`
				StopCount  int  `json:"stop_count"`
				Pricing    struct {
					Total    int    `json:"total"`
					Currency string `json:"currency"`
					FareType string `json:"fare_type"`
				} `json:"pricing"`
				SeatsLeft int    `json:"seats_left"`
				PlaneType string `json:"plane_type"`
				Services  struct {
					WifiAvailable bool `json:"wifi_available"`
					MealsIncluded bool `json:"meals_included"`
					Baggage       struct {
						Cabin string `json:"cabin"`
						Hold  string `json:"hold"`
					} `json:"baggage_allowance"`
				} `json:"services"`
			} `json:"available_flights"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("lion air decode: %w", err)
	}

	flights := make([]entity.Flight, 0, len(resp.Data.AvailableFlights))
	for _, f := range resp.Data.AvailableFlights {
		departAt, err := parseTimeInLocation(f.Schedule.Departure, "2006-01-02T15:04:05", f.Schedule.DepartureTimezone)
		if err != nil {
			return nil, fmt.Errorf("lion air departure time: %w", err)
		}
		arriveAt, err := parseTimeInLocation(f.Schedule.Arrival, "2006-01-02T15:04:05", f.Schedule.ArrivalTimezone)
		if err != nil {
			return nil, fmt.Errorf("lion air arrival time: %w", err)
		}

		aircraft := strings.TrimSpace(f.PlaneType)
		var aircraftPtr *string
		if aircraft != "" {
			aircraftPtr = &aircraft
		}

		amenities := make([]string, 0, 2)
		if f.Services.WifiAvailable {
			amenities = append(amenities, "wifi")
		}
		if f.Services.MealsIncluded {
			amenities = append(amenities, "meal")
		}

		stops := 0
		if !f.IsDirect {
			stops = f.StopCount
			if stops == 0 {
				stops = 1
			}
		}

		duration := durationMinutes(departAt, arriveAt, f.FlightTime)
		flights = append(flights, entity.Flight{
			ID:             fmt.Sprintf("%s_%s", f.ID, l.Name()),
			Provider:       l.Name(),
			Airline:        entity.Airline{Name: f.Carrier.Name, Code: f.Carrier.IATA},
			FlightNumber:   f.ID,
			Departure:      entity.FlightPoint{Airport: f.Route.From.Code, City: f.Route.From.City, Time: departAt},
			Arrival:        entity.FlightPoint{Airport: f.Route.To.Code, City: f.Route.To.City, Time: arriveAt},
			DurationMinute: duration,
			Stops:          stops,
			Price:          entity.Price{Amount: f.Pricing.Total, Currency: f.Pricing.Currency},
			AvailableSeats: f.SeatsLeft,
			CabinClass:     strings.ToLower(f.Pricing.FareType),
			Aircraft:       aircraftPtr,
			Amenities:      amenities,
			Baggage: entity.Baggage{
				CarryOn: f.Services.Baggage.Cabin,
				Checked: f.Services.Baggage.Hold,
			},
		})
	}

	return flights, nil
}
