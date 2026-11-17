-- Initial schema (D-07). Times are UTC Unix milliseconds.
CREATE TABLE tasks (
    id                   INTEGER PRIMARY KEY,
    name                 TEXT    NOT NULL CHECK (length(name) > 0),
    status               TEXT    NOT NULL CHECK (status IN ('active', 'completed', 'cancelled')),
    started_at           INTEGER NOT NULL,
    ended_at             INTEGER,
    original_estimate_ms INTEGER NOT NULL CHECK (original_estimate_ms > 0),
    CHECK ((status = 'active') = (ended_at IS NULL))
);

-- At most one active task (R-03).
CREATE UNIQUE INDEX tasks_one_active ON tasks (status) WHERE status = 'active';
CREATE INDEX tasks_started_at ON tasks (started_at);

CREATE TABLE extensions (
    id       INTEGER PRIMARY KEY,
    task_id  INTEGER NOT NULL REFERENCES tasks (id),
    at       INTEGER NOT NULL,
    added_ms INTEGER NOT NULL
);
CREATE INDEX extensions_task_id ON extensions (task_id);

CREATE TABLE pauses (
    id         INTEGER PRIMARY KEY,
    task_id    INTEGER NOT NULL REFERENCES tasks (id),
    started_at INTEGER NOT NULL,
    ended_at   INTEGER
);
CREATE INDEX pauses_task_id ON pauses (task_id);
