package entity

import "time"

type Airline struct {
	Name string
	Code string
}

type FlightPoint struct {
	Airport string
	City    string
	Time    time.Time
}

type Price struct {
	Amount   int
	Currency string
}

type Baggage struct {
	CarryOn string
	Checked string
}

type Flight struct {
	ID             string
	Provider       string
	Airline        Airline
	FlightNumber   string
	Departure      FlightPoint
	Arrival        FlightPoint
	DurationMinute int
	Stops          int
	Price          Price
	AvailableSeats int
	CabinClass     string
	Aircraft       *string
	Amenities      []string
	Baggage        Baggage
	BestValueScore float64
}
