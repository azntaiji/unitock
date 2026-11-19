-- User settings (D-16), one row per key. The alarm sound is `alarm_sound`.
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
