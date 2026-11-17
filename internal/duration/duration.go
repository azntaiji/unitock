// Package duration parses the natural-language task durations described in D-08.
//
// Accepted forms (case-insensitive, surrounding spaces ignored):
//
//   - one or more <number><unit> terms, optionally separated by spaces, with units
//     h|hr|hrs|hour|hours, m|min|mins|minute|minutes, s|sec|secs|second|seconds, each
//     unit at most once and in h-m-s order ("1h30m", "1 hr 30 min", "2.25 hours");
//   - a bare number, read as minutes ("90", "1.5");
//   - H:MM or H:MM:SS ("1:30", "0:45:30").
//
// The result is rounded to the nearest second and must lie between Min and Max
// inclusive.
package duration

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Bounds of a valid duration, inclusive.
const (
	Min = time.Minute
	Max = 24 * time.Hour
)

// Forms names the accepted input forms, for error messages shown to the user.
const Forms = "use e.g. 30m, 45 minutes, 1h30m, 1 hr 30 min, 90 (minutes) or 1:30"

// maxInputLen bounds the trimmed input so pathological strings fail fast.
const maxInputLen = 64

// Errors returned by Parse. Their messages are suitable for display.
var (
	ErrSyntax = errors.New("unrecognised duration; " + Forms)
	ErrRange  = errors.New("duration must be from 1 minute to 24 hours; " + Forms)
)

type unit int

const (
	hours unit = iota
	minutes
	seconds
)

var unitNames = map[string]unit{
	"h": hours, "hr": hours, "hrs": hours, "hour": hours, "hours": hours,
	"m": minutes, "min": minutes, "mins": minutes, "minute": minutes, "minutes": minutes,
	"s": seconds, "sec": seconds, "secs": seconds, "second": seconds, "seconds": seconds,
}

var unitSeconds = [...]int64{hours: 3600, minutes: 60, seconds: 1}

// Parse converts s to a duration rounded to whole seconds. It returns an error wrapping
// ErrSyntax for malformed input and ErrRange for a well-formed value outside Min..Max.
func Parse(s string) (time.Duration, error) {
	s = asciiLower(strings.TrimSpace(s))
	if s == "" || len(s) > maxInputLen {
		return 0, ErrSyntax
	}

	var total *big.Rat
	var err error
	switch {
	case strings.Contains(s, ":"):
		total, err = parseClock(s)
	case isNumber(s):
		total, err = parseNumber(s)
		if err == nil {
			total.Mul(total, big.NewRat(unitSeconds[minutes], 1))
		}
	default:
		total, err = parseTerms(s)
	}
	if err != nil {
		return 0, err
	}

	secs := roundHalfUp(total)
	if !secs.IsInt64() {
		return 0, ErrRange
	}
	d := time.Duration(secs.Int64()) * time.Second
	if secs.Int64() < int64(Min/time.Second) || secs.Int64() > int64(Max/time.Second) {
		return 0, ErrRange
	}
	return d, nil
}

// parseClock handles H:MM and H:MM:SS.
func parseClock(s string) (*big.Rat, error) {
	parts := strings.Split(s, ":")
	if len(parts) > 3 || !isDigits(parts[0]) {
		return nil, ErrSyntax
	}
	for _, p := range parts[1:] {
		if len(p) != 2 || !isDigits(p) || p[0] > '5' {
			return nil, ErrSyntax
		}
	}
	total := new(big.Rat)
	for i, p := range parts {
		n, ok := new(big.Int).SetString(p, 10)
		if !ok {
			return nil, ErrSyntax
		}
		total.Add(total, new(big.Rat).SetInt(n.Mul(n, big.NewInt(unitSeconds[i]))))
	}
	return total, nil
}

// parseTerms handles one or more <number><unit> terms.
func parseTerms(s string) (*big.Rat, error) {
	total := new(big.Rat)
	next := hours // the earliest unit still allowed
	i := 0
	for i < len(s) {
		start := i
		for i < len(s) && (isDigit(s[i]) || s[i] == '.') {
			i++
		}
		num, err := parseNumber(s[start:i])
		if err != nil {
			return nil, err
		}
		i = skipSpace(s, i)
		start = i
		for i < len(s) && s[i] >= 'a' && s[i] <= 'z' {
			i++
		}
		u, ok := unitNames[s[start:i]]
		if !ok || u < next {
			return nil, ErrSyntax
		}
		next = u + 1
		total.Add(total, num.Mul(num, big.NewRat(unitSeconds[u], 1)))
		i = skipSpace(s, i)
	}
	return total, nil
}

// parseNumber accepts digits with an optional fractional part: "2", "2.25".
func parseNumber(s string) (*big.Rat, error) {
	if !isNumber(s) {
		return nil, ErrSyntax
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("%w: bad number %q", ErrSyntax, s)
	}
	return r, nil
}

func isNumber(s string) bool {
	whole, frac, hasDot := strings.Cut(s, ".")
	return isDigits(whole) && (!hasDot || isDigits(frac))
}

// roundHalfUp returns floor(r + 1/2) for non-negative r.
func roundHalfUp(r *big.Rat) *big.Int {
	x := new(big.Rat).Add(r, big.NewRat(1, 2))
	return new(big.Int).Quo(x.Num(), x.Denom())
}

func skipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// asciiLower lowercases A-Z only, so non-ASCII letters never fold into unit names.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
