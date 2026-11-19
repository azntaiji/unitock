# Decisions register

Status values: `Settled` (fixed by the scaffold or the spec), `Proposed` (recommended;
needs written approval), `Open` (needs information first), `Approved`, `Rejected`,
`Superseded`.

**Coding agent:** build only on `Settled` or `Approved`. Stop and ask on anything else.

To approve, a human changes the status line to `Approved — <name>, <YYYY-MM-DD>`.

`Rejected` and `Superseded` entries move to `archive.md` at gate close, leaving a
one-line pointer here.

Entry shape: **Status**, **Decision**, **Why**, **Alternatives**, **Consequences**.

---

## S-01 — Gate and version approval are human-only; agents tag approved releases
**Status:** Settled
Agents request gates and propose versions. Humans approve in `GATES.md`, by hand or by
confirming the exact line the agent proposes after an explicit reply. At gate close
the agent commits the release, creates the annotated tag `vX.Y.Z` for the approved
version, and pushes it after `yeet` pushes the branch. Tags are never moved or deleted.

## S-02 — Semantic Versioning 2.0.0 with `VERSION` as the source of truth
**Status:** Settled
See `docs/process/versioning.md`.

## S-03 — Agents commit locally with Conventional Commits 1.0.0; pushes go through `yeet`
**Status:** Settled
Agents commit at each task boundary and never push directly. At gate close the agent
invokes the `yeet` skill, which asks the human to confirm before it commits and pushes.
The one direct push an agent makes is the release tag (S-01), and only after `yeet` has
pushed the release commit. The human can also run `/yeet` at any time for a remote
backup.

---

## D-01 — Language profile
**Status:** Settled
**Decision:** Go (`profiles/go/`), toolchain and commands per its PROFILE.md. Module path
`github.com/azntaiji/unitock`; binary `unitock`.
**Why:** specified by the human in the app idea (Go + Bubble Tea, macOS and Ubuntu).

## D-02 — License and repository visibility
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** MIT license; public GitHub repository. (Human's answer in the Phase 0
interview, 2026-11-17; RFI-01.)

## D-03 — Branching model
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** trunk-based. Agents commit locally to the default branch at task
boundaries; the remote receives a phase when `yeet` runs at gate close.
**Alternatives:** one branch per phase, merged by pull request at gate close.
**Consequences:** work in progress exists only on the working machine until gate close. Run `/yeet` mid-phase for a backup; releases are still marked only by tags.

## D-04 — TUI stack
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** Bubble Tea v2, Bubbles v2 (textinput, progress), and Lip Gloss v2 from
Charm, latest stable at Phase 1 start. TUI flows tested with Charm's `teatest`, views
with golden files. Big digits are a small in-repo glyph table, not a dependency.
**Why:** the human named Bubble Tea; v2 is the current major line.
**Alternatives:** Bubble Tea v1 (older API, maintenance only).
**Consequences:** `teatest` lives in `x/exp` and may change; pin its version.

## D-05 — SQLite driver
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** `modernc.org/sqlite` (pure Go) via `database/sql`.
**Why:** the release kit builds with `CGO_ENABLED=0` for four OS/arch targets (D-06).
**Alternatives:** `mattn/go-sqlite3` (cgo; needs a C cross-toolchain per target);
`ncruces/go-sqlite3` (Wasm-based, pure Go, younger).
**Consequences:** larger binary (~+8 MB) and somewhat slower queries; irrelevant at this
data size.

## D-06 — Release kit
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** adopt the Go profile's GoReleaser release kit. Each approved `vX.Y.Z` tag
publishes darwin/linux × amd64/arm64 binaries to GitHub Releases; `go install
github.com/azntaiji/unitock/cmd/unitock@latest` also works. `make release-check` is a
required check at every gate that proposes a version. (Human's answer in the Phase 0
interview, 2026-11-17.)

## D-07 — Data and time model
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** times stored as UTC Unix milliseconds. Tables:
`tasks(id, name, status, started_at, ended_at, original_estimate_ms)`,
`extensions(id, task_id, at, added_ms)`, `pauses(id, task_id, started_at, ended_at)`.
A partial unique index allows one `active` task. Estimate = original + sum(extensions);
elapsed = (ended_at or now) − started_at − paused time (an open pause counts to now).
Nothing is ticked or accumulated in memory; all state is derived from stored timestamps
and an injected clock. Migrations are numbered SQL files embedded in the binary, applied
in a transaction at startup, tracked with `PRAGMA user_version`. WAL mode.
**Why:** makes R-04/R-05 (resume, sleep, crash) fall out of the model, and keeps timer
maths pure and testable.
**Alternatives:** storing a running elapsed counter (drifts, loses time on crash).
**Consequences:** changing the clock on the machine changes elapsed time; accepted.

## D-08 — Duration grammar
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** case-insensitive, surrounding spaces ignored. Accepted forms:
(a) one or more `<number><unit>` terms with optional spaces between them, number
integer or decimal, units `h|hr|hrs|hour|hours`, `m|min|mins|minute|minutes`,
`s|sec|secs|second|seconds`, each unit at most once, in h-m-s order;
(b) a bare number = minutes; (c) `H:MM` (or `H:MM:SS`). Result rounded to the nearest
second; valid range 1 minute to 24 hours inclusive. Fuzz-tested.
**Alternatives:** Go's `time.ParseDuration` only (rejects "45 minutes", "1hr").

## D-09 — Timer visuals and colour thresholds
**Status:** Approved — azntaiji, 2026-11-17 (amended and re-approved the same day)
**Decision:** large 5-row block digits (plain bold below 60 columns of free width), a
horizontal progress bar with elapsed percentage, and a status badge. All colours are the
terminal theme's 16-colour ANSI palette, never fixed RGB values. Digits and bar colour by
remaining fraction of the estimate: > 50% default foreground; ≤ 50% blue; ≤ 25% yellow;
≤ 10% and overtime red. Paused dims the digits. The badge is bold, black text on a
coloured background, one of: `● RUNNING` (green) above 25% left; `◐ ENDING` (yellow) at
25% or less; `▲ OVERTIME` (red) past the estimate; `❚❚ PAUSED` (white, ANSI 7) whenever
paused, overriding the others. Under `NO_COLOR` the badge renders in reverse video with
the same text.
**Why:** ANSI colours follow the user's theme; green and light grey keep black text
readable where ANSI blue and bright black are dark in many themes.
**Alternatives:** a ring drawn with braille characters (harder to read at small sizes).

## D-10 — Extending in overtime
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** when Extend is chosen from the time-up alert, the estimate grows by the
overtime so far plus the entered amount, so remaining equals the entered amount from now.
Extending before time is up (`d`) adds exactly the entered amount. Each extension is
stored as its own row with the amount actually added.
**Why:** "give me 10 more minutes" should mean 10 minutes from now, not 10 minus however
long the alarm rang.
**Alternatives:** always add only the entered amount (alert can re-fire at once).
**Consequences:** time spent ignoring the alarm counts toward the estimate in stats.

## D-11 — Stats periods and buckets
**Status:** Approved — azntaiji, 2026-11-18
**Decision:** all dates local. 7 days = today and the 6 days before, 7 daily buckets.
30 days = today and the 29 days before, buckets of 7 days starting at the period's first
day (the last bucket has 2 days), labelled by bucket start. 3 months = from the 1st of
the month two months ago through today, one bucket per calendar month (current month
partial). Previous period = the same-length window immediately before (for 3 months, the
three preceding calendar months). Average per day = completed ÷ days in the period
(elapsed days for 3 months). Week/month boundaries follow the local time zone, with DST
handled by calendar dates, not 24-hour steps.
**Alternatives:** calendar weeks for 30 days; a rolling 90 days for 3 months (would give
4 partial month bars).

## D-12 — Single-instance lock
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** an exclusive advisory `flock` on `unitock.lock` in the data directory, held
for the life of the process (released by the OS on crash).
**Alternatives:** PID file (stale after crashes); rely on SQLite locking (two UIs would
fight over the active task).

## D-13 — Alarm sound and notifications
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** the chime is a short WAV synthesised by the app's own code (no third-party
audio asset, no licensing question), written once to the data directory. It loops by
re-running the first available player: `afplay` on macOS; `paplay`, then `aplay` on
Linux; stopped by killing the process. Fallback: terminal bell every 2 s. Notification:
`osascript -e 'display notification …'` on macOS (task name passed as an argument via
`on run argv`, never interpolated), `notify-send` on Linux. Failures are logged, never
fatal. Behind an interface so tests use a fake.
**Alternatives:** an audio library such as `oto` (needs cgo on Linux, breaks D-05/D-06).

## D-14 — Export format
**Status:** Approved — azntaiji, 2026-11-18
**Decision:** JSON schema version 1 exactly as in `docs/SPEC.md` Interfaces. Fields are
only ever added in later 1.x versions; renames or removals bump `schema_version` and are
breaking changes.

## D-15 — Projects
**Status:** Approved — azntaiji, 2026-11-17
**Decision:** migration 0002 adds `projects(id, name UNIQUE COLLATE NOCASE, created_at)`
and a nullable `tasks.project_id` referencing it. Starting a task with a project name
reuses the case-insensitive match or creates the project, in the same transaction as the
task. Suggestions (R-26) rank prefix matches before other substring matches, then by the
latest `started_at` of the project's tasks (projects with no tasks by `created_at`).
**Why:** one row per project keeps spellings consistent and suggestions cheap.
**Alternatives:** a free-text `tasks.project` column with `SELECT DISTINCT` (case
variants drift apart).
**Consequences:** amends D-07's table list; export emits the project name or null.
**Amended:** azntaiji, 2026-11-18. SQLite `NOCASE` folds ASCII only, so "über" and
"Über" became two projects. Migration 0003 adds `projects.name_key`, the name with full
Unicode simple case folding applied in Go (the same rule as `strings.EqualFold`), with a
unique index; the store matches on it instead of `NOCASE`. On upgrade the store fills
the keys for existing rows; where two existing projects share a key, the older one is
kept and the other's tasks move to it.

## D-16 — Choosable alarm sounds
**Status:** Approved — azntaiji, 2026-11-19 (in conversation, on the agent's recommendation)
**Decision:** five alarm sounds, all synthesised by the app's own code like D-13's chime:
Chime (the D-13 chime, default), Bell, Beep, Marimba, and Rising. The choice is made on a
Sound screen opened with `a` from Start (R-51 to R-53), which previews each sound once
as it is highlighted, and is stored in the database: migration 0005 adds
`settings(key TEXT PRIMARY KEY, value TEXT NOT NULL)`, key `alarm_sound`, value the
sound's lower-case name. The alarm writes the chosen sound to `alarm.wav` (rewritten
only when it differs) and plays it with D-13's players. A preview plays once and never
delays or replaces the time-up alarm.
**Why:** picking a sound needs to hear it, which a menu with preview gives and a config
file does not; a configuration file is a SPEC non-goal, and the database already holds
the user's data. Synthesised sounds need no assets or licences.
**Alternatives:** a config file or `--sound` flag (non-goal; no preview); shipping
recorded audio files (licensing, binary size); a silent option (breaks the "unmissable
alert" goal).
**Consequences:** amends D-07's table list and D-13's single chime; R-14, R-34 amended;
the Ubuntu end-to-end check must hear every sound through `paplay` and `aplay`.
**Amended:** azntaiji, 2026-11-19. The Sound screen opens with `ctrl+o` from any focus
on Start, not `a` on the button; Start shows `Alarm: <name>` beside the button. Part of
making app-wide commands `ctrl` chords only (R-06): `ctrl+s` stats, `ctrl+c` quit.
