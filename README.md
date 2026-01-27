# BookCabin Flight Search Aggregator

Flight search and aggregation service that normalizes mock provider data, filters and sorts results, and returns a unified response.

## Features
- Aggregates data from Garuda Indonesia, Lion Air, Batik Air, and AirAsia mock providers.
- Normalizes data into a unified flight response structure.
- Filters by price range, stops, airlines, duration, and departure/arrival times.
- Sorts by price, duration, departure, arrival, or best value.
- Handles mixed time formats and time zones.
- Adds caching and provider retry logic for temporary failures.
- Supports round-trip searches with `return_date`.
- Formats IDR prices with thousands separators in responses.
- Applies per-provider rate limiting.
- Compares prices across providers for the same flight and keeps the lowest fare.

## How To Run
- Prerequisite: Go 1.25+
- `cp config/config.example.yaml config/config.yaml`
- `make run`
- Or: `LOCAL=true go run main.go`
- Server listens on `0.0.0.0:8080` by default.

## API Usage

Search Flights:
```bash
curl "http://localhost:8080/flights?origin=CGK&destination=DPS&departureDate=2025-12-15&passengers=1&cabinClass=economy"
```

Round-trip search:
```bash
curl "http://localhost:8080/flights?origin=CGK&destination=DPS&departureDate=2025-12-15&return_date=2025-12-20&passengers=1&cabinClass=economy"
```

Response note:
- `return_flights` is included when `return_date` is provided.
- `price.formatted` includes IDR formatting (e.g., `Rp. 1.250.000`).

Optional filters:
- `min_price`, `max_price`
- `stops` (exact), `max_stops`
- `min_duration`, `max_duration` (minutes)
- `depart_after`, `depart_before`, `arrive_after`, `arrive_before` (RFC3339 or HH:MM)
- `airlines` (comma-separated names or IATA codes)
- `sort` (`price`, `duration`, `departure`, `arrival`, `best_value`)
- `order` (`asc`, `desc`)

## Mock Providers
Mock JSON fixtures live in `mocks/` and are loaded at runtime:
- `mocks/garuda_indonesia_search_response.json`
- `mocks/lion_air_search_response.json`
- `mocks/batik_air_search_response.json`
- `mocks/airasia_search_response.json`
These fixtures include example return-leg flights for DPS -> CGK on `2025-12-20` to exercise round-trip searches.

## Design Notes
- Providers are queried in parallel with per-provider timeouts.
- AirAsia has a 90% success rate and uses exponential backoff retries.
- Cache TTL defaults to 60 seconds per search criteria + filters.
- Best value score combines normalized price (60%) and duration (40%).

## Design Choices
- Used provider adapters to normalize diverse response formats into a single entity model.
- Kept aggregation, filtering, and sorting in the usecase layer for separation of concerns.
- Added a small in-memory cache to reduce repeated provider calls during short windows.
- Rate limiting is applied per provider instance to mimic external API constraints.

## Implementation Details
- Aggregation queries providers in parallel with timeouts, then validates and filters results.
- Duration is recalculated from timestamps when possible to include layovers.
- Best value scoring uses normalized price and duration to keep results stable across providers.
- Round-trip searches run a second query with reversed origin/destination and adjusted filters.
- Response includes normalized timestamps, formatted durations, and formatted IDR pricing.
- Price comparison deduplicates flights by airline/flight number and timestamps.

## Configuration
- `modules.book-cabin.cache.ttl_seconds`: cache TTL in seconds (default 60).
- `modules.book-cabin.provider.rate_limit_ms`: minimum delay between requests per provider (default 100ms).

## Not Implemented
Required
- None.

Bonus
- Explicit WIB/WITA/WIT label parsing BUT **(offsets/time zone names are supported)**.
- Multi-city searches.
