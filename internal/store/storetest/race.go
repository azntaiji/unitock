//go:build race

package storetest

// Race reports whether the race detector is on. Timing checks skip under it, since it
// slows code several times over.
const Race = true
