// Package timeutil holds shared time helpers. Its job today is a single source
// of truth for the business timezone used to bucket day-grained figures, so the
// order and wallet KPIs agree on where "today" starts.
package timeutil

import (
	"os"
	"time"
	_ "time/tzdata" // embed the IANA tz database so BUSINESS_TZ resolves regardless of the host OS
)

// businessLocation is resolved once at init from BUSINESS_TZ.
var businessLocation = loadBusinessLocation()

func loadBusinessLocation() *time.Location {
	if name := os.Getenv("BUSINESS_TZ"); name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	return time.UTC
}

// BusinessLocation is the timezone used to bucket day-grained figures ("today"
// revenue/orders, wallet top-ups today, the daily revenue chart). It defaults to
// UTC; set BUSINESS_TZ to an IANA name (e.g. "Asia/Beirut") so day boundaries
// line up with the operating market rather than UTC. An unparseable value falls
// back to UTC so nothing fails to load.
func BusinessLocation() *time.Location { return businessLocation }
