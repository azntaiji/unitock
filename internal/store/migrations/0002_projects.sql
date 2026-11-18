-- Optional projects (D-15, R-25). Names are unique ignoring case.
CREATE TABLE projects (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE COLLATE NOCASE CHECK (length(name) > 0),
    created_at INTEGER NOT NULL
);

ALTER TABLE tasks ADD COLUMN project_id INTEGER REFERENCES projects (id);
CREATE INDEX tasks_project_id ON tasks (project_id);
