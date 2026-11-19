package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionFromBuildInfo(t *testing.T) {
	for _, tc := range []struct {
		name    string
		info    *debug.BuildInfo
		ok      bool
		v, c, d string
	}{
		{"no build info", nil, false, "dev", "none", "unknown"},
		{"test binary", &debug.BuildInfo{}, true, "dev", "none", "unknown"},
		{"devel build", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true, "dev", "none", "unknown"},
		{"go install from the proxy", &debug.BuildInfo{Main: debug.Module{Version: "v1.0.0"}}, true, "1.0.0", "none", "unknown"},
		{"local build in a checkout", &debug.BuildInfo{
			Main: debug.Module{Version: "v1.0.1-0.20261119184715-cef31d6a1b2c+dirty"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "cef31d6a1b2c3d4e5f60718293a4b5c6d7e8f901"},
				{Key: "vcs.time", Value: "2026-11-19T18:47:15Z"},
				{Key: "GOOS", Value: "linux"},
			},
		}, true, "1.0.1-0.20261119184715-cef31d6a1b2c+dirty", "cef31d6", "2026-11-19T18:47:15Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, c, d := fromBuildInfo(tc.info, tc.ok, "dev", "none", "unknown")
			if v != tc.v || c != tc.c || d != tc.d {
				t.Errorf("got %q %q %q, want %q %q %q", v, c, d, tc.v, tc.c, tc.d)
			}
		})
	}
}
