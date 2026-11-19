package store

import (
	"context"
	"testing"
)

func TestSoundSettingRoundTripsAndSurvivesReopen(t *testing.T) {
	ctx := context.Background()
	s, dir := openTemp(t)
	if v, ok, err := s.Setting(ctx, SettingAlarmSound); err != nil || ok || v != "" {
		t.Fatalf("fresh database: %q, %v, %v; want none", v, ok, err)
	}
	for _, v := range []string{"bell", "rising"} { // the second replaces the first
		if err := s.SetSetting(ctx, SettingAlarmSound, v); err != nil {
			t.Fatal(err)
		}
	}
	_ = s.Close()

	s2, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s2.Close() }()
	if v, ok, err := s2.Setting(ctx, SettingAlarmSound); err != nil || !ok || v != "rising" {
		t.Errorf("after reopen: %q, %v, %v; want rising", v, ok, err)
	}
	if _, ok, _ := s2.Setting(ctx, "other"); ok {
		t.Error("an unset key reported a value")
	}
}

func TestSoundSettingErrorOnClosedStore(t *testing.T) {
	s, _ := openTemp(t)
	_ = s.Close()
	if err := s.SetSetting(context.Background(), SettingAlarmSound, "bell"); err == nil {
		t.Error("SetSetting on a closed store succeeded")
	}
	if _, _, err := s.Setting(context.Background(), SettingAlarmSound); err == nil {
		t.Error("Setting on a closed store succeeded")
	}
}
