package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SettingAlarmSound is the settings key for the chosen alarm sound (D-16, R-53).
const SettingAlarmSound = "alarm_sound"

// Setting returns the value stored under key, or ok false if there is none.
func (s *Store) Setting(ctx context.Context, key string) (value string, ok bool, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("reading setting %s: %w", key, err)
	}
	return value, true, nil
}

// SetSetting stores value under key, replacing any earlier value.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value`,
		key, value); err != nil {
		return fmt.Errorf("saving setting %s: %w", key, err)
	}
	return nil
}
