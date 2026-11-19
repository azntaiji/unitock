package main

import (
	"runtime/debug"
	"strings"
)

// Release builds set version, commit, and date with -ldflags. Other builds, such as
// `go install …@v1.0.0`, are left with "dev"; for those, take what the Go toolchain
// recorded in the binary instead, so --version and exports name the real version
// (SPEC Interfaces, D-06).
func init() {
	if version == "dev" {
		info, ok := debug.ReadBuildInfo()
		version, commit, date = fromBuildInfo(info, ok, version, commit, date)
	}
}

// fromBuildInfo returns the module version (without its leading "v") and, when the
// build recorded them, the short VCS revision and commit time. Anything missing keeps
// its given default: a test binary or a build outside a module has no version.
func fromBuildInfo(info *debug.BuildInfo, ok bool, v, c, d string) (string, string, string) {
	if !ok || info == nil {
		return v, c, d
	}
	if mv := info.Main.Version; mv != "" && mv != "(devel)" {
		v = strings.TrimPrefix(mv, "v")
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			c = s.Value[:min(7, len(s.Value))]
		case "vcs.time":
			d = s.Value
		}
	}
	return v, c, d
}
