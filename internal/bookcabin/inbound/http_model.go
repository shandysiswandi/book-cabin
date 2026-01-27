package inbound

type FlightsResponse struct {
	SearchCriteria SearchCriteriaResponse `json:"search_criteria"`
	Metadata       MetadataResponse       `json:"metadata"`
	Flights        []FlightResponse       `json:"flights"`
	ReturnFlights  []FlightResponse       `json:"return_flights,omitempty"`
}

type SearchCriteriaResponse struct {
	Origin        string  `json:"origin"`
	Destination   string  `json:"destination"`
	DepartureDate string  `json:"departure_date"`
	ReturnDate    *string `json:"return_date,omitempty"`
	Passengers    int     `json:"passengers"`
	CabinClass    string  `json:"cabin_class"`
}

type MetadataResponse struct {
	TotalResults       int   `json:"total_results"`
	ProvidersQueried   int   `json:"providers_queried"`
	ProvidersSucceeded int   `json:"providers_succeeded"`
	ProvidersFailed    int   `json:"providers_failed"`
	SearchTimeMs       int64 `json:"search_time_ms"`
	CacheHit           bool  `json:"cache_hit"`
}

type FlightResponse struct {
	ID             string           `json:"id"`
	Provider       string           `json:"provider"`
	Airline        AirlineResponse  `json:"airline"`
	FlightNumber   string           `json:"flight_number"`
	Departure      FlightPoint      `json:"departure"`
	Arrival        FlightPoint      `json:"arrival"`
	Duration       DurationResponse `json:"duration"`
	Stops          int              `json:"stops"`
	Price          PriceResponse    `json:"price"`
	AvailableSeats int              `json:"available_seats"`
	CabinClass     string           `json:"cabin_class"`
	Aircraft       *string          `json:"aircraft"`
	Amenities      []string         `json:"amenities"`
	Baggage        BaggageResponse  `json:"baggage"`
}

type AirlineResponse struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type FlightPoint struct {
	Airport   string `json:"airport"`
	City      string `json:"city"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

type DurationResponse struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type PriceResponse struct {
	Amount    int    `json:"amount"`
	Currency  string `json:"currency"`
	Formatted string `json:"formatted"`
}

type BaggageResponse struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}
