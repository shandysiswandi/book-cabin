package provider

import (
	"fmt"
	"strings"
	"time"
)

func cityFromAirport(code string) string {
	switch strings.ToUpper(code) {
	case "CGK":
		return "Jakarta"
	case "DPS":
		return "Denpasar"
	case "SUB":
		return "Surabaya"
	case "UPG":
		return "Makassar"
	case "SOC":
		return "Solo"
	default:
		return ""
	}
}

func parseTimeWithLayout(value, layout string) (time.Time, error) {
	return time.Parse(layout, value)
}

func parseTimeInLocation(value, layout, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load location %s: %w", timezone, err)
	}
	return time.ParseInLocation(layout, value, loc)
}

func durationMinutes(depart, arrive time.Time, fallback int) int {
	if depart.IsZero() || arrive.IsZero() {
		return fallback
	}
	diff := int(arrive.Sub(depart).Minutes())
	if diff <= 0 {
		return fallback
	}
	return diff
}
