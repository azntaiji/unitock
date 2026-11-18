-- Stats read completed tasks by the time they ended (R-44).
CREATE INDEX tasks_ended_at ON tasks (ended_at);
