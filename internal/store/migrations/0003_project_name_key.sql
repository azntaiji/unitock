-- Unicode case-insensitive project names (amended D-15, R-25). name_key is the name
-- folded in Go (foldKey); the store fills it for existing rows, merges projects that
-- share a key, and then creates the unique index (see goSteps in store.go).
ALTER TABLE projects ADD COLUMN name_key TEXT;
