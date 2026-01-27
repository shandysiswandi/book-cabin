package inbound

import (
	"context"
	"net/http"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/entity"
)

type HTTPEndpoint struct {
	uc uc
}

func (h *HTTPEndpoint) Flights(ctx context.Context, r *http.Request) (any, error) {
	input, err := parseFlightsInput(r)
	if err != nil {
		return nil, err
	}

	output, err := h.uc.Flights(ctx, input)
	if err != nil {
		return nil, err
	}

	flights := mapFlightResponses(output.Flights)
	returnFlights := mapFlightResponses(output.ReturnFlights)

	return FlightsResponse{
		SearchCriteria: SearchCriteriaResponse{
			Origin:        output.SearchCriteria.Origin,
			Destination:   output.SearchCriteria.Destination,
			DepartureDate: output.SearchCriteria.DepartureDate,
			ReturnDate:    output.SearchCriteria.ReturnDate,
			Passengers:    output.SearchCriteria.Passengers,
			CabinClass:    output.SearchCriteria.CabinClass,
		},
		Metadata: MetadataResponse{
			TotalResults:       output.Metadata.TotalResults,
			ProvidersQueried:   output.Metadata.ProvidersQueried,
			ProvidersSucceeded: output.Metadata.ProvidersSucceeded,
			ProvidersFailed:    output.Metadata.ProvidersFailed,
			SearchTimeMs:       output.Metadata.SearchTimeMs,
			CacheHit:           output.Metadata.CacheHit,
		},
		Flights:       flights,
		ReturnFlights: returnFlights,
	}, nil
}

func mapFlightResponses(flights []entity.Flight) []FlightResponse {
	resp := make([]FlightResponse, 0, len(flights))
	for _, flight := range flights {
		resp = append(resp, FlightResponse{
			ID:             flight.ID,
			Provider:       flight.Provider,
			Airline:        AirlineResponse{Name: flight.Airline.Name, Code: flight.Airline.Code},
			FlightNumber:   flight.FlightNumber,
			Departure:      mapFlightPoint(flight.Departure),
			Arrival:        mapFlightPoint(flight.Arrival),
			Duration:       DurationResponse{TotalMinutes: flight.DurationMinute, Formatted: formatDuration(flight.DurationMinute)},
			Stops:          flight.Stops,
			Price:          PriceResponse{Amount: flight.Price.Amount, Currency: flight.Price.Currency, Formatted: formatIDR(flight.Price.Amount)},
			AvailableSeats: flight.AvailableSeats,
			CabinClass:     flight.CabinClass,
			Aircraft:       flight.Aircraft,
			Amenities:      append([]string{}, flight.Amenities...),
			Baggage:        BaggageResponse{CarryOn: flight.Baggage.CarryOn, Checked: flight.Baggage.Checked},
		})
	}
	return resp
}
