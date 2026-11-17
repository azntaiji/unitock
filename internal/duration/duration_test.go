package duration

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseAccepts(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		// Every example in SPEC R-10.
		{"30m", 30 * time.Minute},
		{"45 minutes", 45 * time.Minute},
		{"1hr", time.Hour},
		{"2.25 hours", 2*time.Hour + 15*time.Minute},
		{"1h30m", 90 * time.Minute},
		{"1 hr 30 min", 90 * time.Minute},
		{"90", 90 * time.Minute},
		{"1:30", 90 * time.Minute},

		// Case and surrounding space.
		{"  30M  ", 30 * time.Minute},
		{"1 HOUR", time.Hour},
		{"\t2Hrs\n", 2 * time.Hour},

		// Every unit spelling.
		{"1h", time.Hour}, {"1hrs", time.Hour}, {"1hour", time.Hour}, {"1 hours", time.Hour},
		{"5m", 5 * time.Minute}, {"5min", 5 * time.Minute}, {"5mins", 5 * time.Minute},
		{"5minute", 5 * time.Minute}, {"5 minutes", 5 * time.Minute},
		{"90s", 90 * time.Second}, {"90sec", 90 * time.Second}, {"90secs", 90 * time.Second},
		{"90second", 90 * time.Second}, {"90 seconds", 90 * time.Second},

		// Combinations in h-m-s order.
		{"1h30m15s", 90*time.Minute + 15*time.Second},
		{"1h 15s", time.Hour + 15*time.Second},
		{"2m 30s", 2*time.Minute + 30*time.Second},
		{"1 hour 2 minutes 3 seconds", time.Hour + 2*time.Minute + 3*time.Second},
		{"1hr30min", 90 * time.Minute},
		{"1h\t30m", 90 * time.Minute},

		// Decimals and bare numbers.
		{"1.5", 90 * time.Second},
		{"1.5h", 90 * time.Minute},
		{"0.1h", 6 * time.Minute},
		{"0.5 min 45s", 75 * time.Second},
		{"007", 7 * time.Minute},

		// Clock forms.
		{"0:01", time.Minute},
		{"0:45:30", 45*time.Minute + 30*time.Second},
		{"24:00", 24 * time.Hour},
		{"10:05:09", 10*time.Hour + 5*time.Minute + 9*time.Second},

		// Rounding to the nearest second, halves up.
		{"1.0083333m", time.Minute}, // 60.499998s
		{"59.5s", time.Minute},
		{"60.4s", time.Minute},
		{"60.5s", 61 * time.Second},
		{"1.00001", time.Minute},

		// Range bounds, inclusive.
		{"1m", Min},
		{"60s", Min},
		{"24h", Max},
		{"1440", Max},
		{"23h 59m 60s", Max},
		{"86400.4s", Max},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseRejects(t *testing.T) {
	tests := []struct {
		in   string
		want error
	}{
		// Malformed.
		{"", ErrSyntax},
		{"   ", ErrSyntax},
		{"abc", ErrSyntax},
		{"h", ErrSyntax},
		{"m30", ErrSyntax},
		{"30x", ErrSyntax},
		{"30 mi", ErrSyntax},
		{"1h30", ErrSyntax},
		{"30m1h", ErrSyntax},
		{"1s 2m", ErrSyntax},
		{"1h 2h", ErrSyntax},
		{"10m 5m", ErrSyntax},
		{"1hm", ErrSyntax},
		{"1..5h", ErrSyntax},
		{"1.h", ErrSyntax},
		{".5h", ErrSyntax},
		{"1.", ErrSyntax},
		{".5", ErrSyntax},
		{"-5m", ErrSyntax},
		{"+5m", ErrSyntax},
		{"1,5h", ErrSyntax},
		{"1e3", ErrSyntax},
		{"1_000", ErrSyntax},
		{"1/2h", ErrSyntax},
		{"1:3", ErrSyntax},
		{"1:300", ErrSyntax},
		{"1:60", ErrSyntax},
		{"1:30:60", ErrSyntax},
		{":30", ErrSyntax},
		{"1:", ErrSyntax},
		{"1:30:", ErrSyntax},
		{"1:00:00:00", ErrSyntax},
		{"1.5:30", ErrSyntax},
		{"1 : 30", ErrSyntax},
		{"1h:30", ErrSyntax},
		{"30 m in", ErrSyntax},
		{"1 h, 30 m", ErrSyntax},
		{"1mın", ErrSyntax}, // dotless i must not fold to "min"
		{"1ſ", ErrSyntax},   // long s must not fold to "s"
		{"30 m", ErrSyntax},
		{strings.Repeat("1", maxInputLen+1), ErrSyntax},

		// Well-formed but out of range.
		{"0", ErrRange},
		{"0m", ErrRange},
		{"59s", ErrRange},
		{"59.4s", ErrRange},
		{"0.5", ErrRange},
		{"0:00", ErrRange},
		{"0:00:59", ErrRange},
		{"24:00:01", ErrRange},
		{"24h 1s", ErrRange},
		{"86400.5s", ErrRange},
		{"1441", ErrRange},
		{"25h", ErrRange},
		{"99999999999999999999999999h", ErrRange},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Parse(%q) = %v, %v; want error %v", tt.in, got, err, tt.want)
			}
			if got != 0 {
				t.Errorf("Parse(%q) returned %v alongside an error", tt.in, got)
			}
		})
	}
}

func TestErrorsNameAcceptedForms(t *testing.T) {
	for _, err := range []error{ErrSyntax, ErrRange} {
		for _, form := range []string{"30m", "45 minutes", "1h30m", "90", "1:30"} {
			if !strings.Contains(err.Error(), form) {
				t.Errorf("%q does not mention %q", err, form)
			}
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"30m", "45 minutes", "1hr", "2.25 hours", "1h30m", "1 hr 30 min", "90", "1:30",
		"0:45:30", "24:00", "59.5s", "1h 2h", "1:60", "", "  1H ", "1.5", "abc",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		d, err := Parse(s)
		if err != nil {
			if !errors.Is(err, ErrSyntax) && !errors.Is(err, ErrRange) {
				t.Fatalf("Parse(%q): unexpected error kind %v", s, err)
			}
			if d != 0 {
				t.Fatalf("Parse(%q) returned %v with an error", s, d)
			}
			return
		}
		if d < Min || d > Max {
			t.Fatalf("Parse(%q) = %v, outside [%v, %v]", s, d, Min, Max)
		}
		if d%time.Second != 0 {
			t.Fatalf("Parse(%q) = %v, not whole seconds", s, d)
		}
		// Case and surrounding spaces must not change the result.
		for _, v := range []string{asciiUpper(s), " " + s + " ", "\t" + s} {
			if d2, err2 := Parse(v); err2 != nil || d2 != d {
				t.Fatalf("Parse(%q) = %v, %v; want %v (from %q)", v, d2, err2, d, s)
			}
		}
	})
}

func asciiUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
	}
	return string(b)
}
