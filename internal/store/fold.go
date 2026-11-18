package store

import (
	"strings"
	"unicode"
)

// foldKey returns the key under which project names are matched (amended D-15): each
// rune is replaced by the smallest rune of its Unicode simple case folding orbit. Two
// names have the same key exactly when strings.EqualFold reports them equal, so "über"
// and "Über" (and "K" and the Kelvin sign) share a key, while "ß" and "ss" do not.
func foldKey(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		low := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			low = min(low, f)
		}
		b.WriteRune(low)
	}
	return b.String()
}
