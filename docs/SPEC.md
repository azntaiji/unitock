# Spec

Written in Phase 0 from an interview with the human (azntaiji, 2026-11-17). A new session
should be able to build from this file, `docs/PLAN.md`, and the decisions in
`docs/decisions/DECISIONS.md` that requirements cite (D-xx). Those decisions hold the
detailed rules (duration grammar, schema, colours, stats buckets, export) and are part
of the spec once Approved.

## Problem

Task managers and Pomodoro apps add projects, tags, and cycles that get in the way of the
one thing wanted here: say what you are working on, say how long it should take, and be
told clearly when the time is up. There is also no record afterwards of how estimates
compared with reality.

## Users and use cases

One person working in a terminal on macOS or Ubuntu. They:

1. Start a task with a name and an estimated duration in plain language ("45m", "1.5 hours").
2. Watch a countdown, pause it, extend it, cancel it, or mark it complete.
3. Get an unmissable alert (on-screen, system notification, looping sound) when time runs out.
4. Review how many tasks they completed and how actual time compared with estimates over
   the last 7 days, 30 days, or 3 months.
5. Export everything as JSON.

## Goals

- Starting a task takes one screen and under 10 seconds of typing.
- The timer is correct to the second across pauses, app restarts, and machine sleep.
- The time-up alert keeps sounding until the user responds.
- No data is lost if the app is quit, killed, or the terminal is closed mid-task.

## Non-goals

- More than one task at a time, task lists, projects beyond an optional project label per
  task, tags, or priorities.
- Editing or deleting past tasks from the UI.
- Pomodoro cycles, breaks, or recurring tasks.
- Sync, accounts, cloud storage, or any network access.
- Windows support, a GUI, a configuration file, localisation, or data import.

## Requirements

Terms: the **estimate** is the original duration plus all extensions. **Elapsed** (also
"actual") is running time since start: wall-clock time minus paused time. **Remaining** is
estimate minus elapsed; below zero it is **overtime**.

### General

- R-01 The app is a full-screen terminal UI named `unitock`, built with Go and Bubble Tea.
- R-02 All data (task names, projects, estimates, extensions, pauses, start and end times, status)
  is stored in one SQLite database. Every state change is written before the UI shows it.
- R-03 At most one task is active at a time. A task is `active`, `completed`, or `cancelled`.
- R-04 Time is wall-clock based (D-07). Time while the app is closed or the machine sleeps
  counts as elapsed unless the task is paused.
- R-05 If a task is active at launch, the app opens on the Active screen with the timer
  where wall-clock time puts it. If the task ran out while the app was closed, the alert
  (R-30) fires immediately. A task paused at quit is still paused at launch.
- R-06 `ctrl+c` quits from any screen and state, including while typing and while the
  alert is open. App-wide commands are `ctrl` chords so they never clash with typing:
  `ctrl+c` quit, `ctrl+s` stats (R-24), `ctrl+o` alarm sound (R-51); plain letters
  never quit. Quitting never ends or changes the active task.
- R-07 Every screen shows a key-hint bar listing the keys that work in its current state.
  Where they do not fit on one line, the screen's own keys come first and the `ctrl`
  chords on a second line.
- R-08 Below a minimum terminal size of 60×20 the app shows a "terminal too small"
  message instead of the screen; it redraws correctly on resize.
- R-09 Only one instance runs per data directory (D-12). A second instance exits with
  status 1 and a message naming the lock file.

### Durations

- R-10 Durations are parsed per D-08. Accepted examples: `30m`, `45 minutes`, `1hr`,
  `2.25 hours`, `1h30m`, `1 hr 30 min`, `90` (bare number = minutes), `1:30` (h:mm).
  Parsing is case-insensitive and ignores surrounding spaces.
- R-11 A valid duration is from 1 minute to 24 hours inclusive, rounded to whole seconds.
  Anything else shows an inline error naming the accepted forms.

### Start screen

- R-12 Three inputs, Project, Task name, and Task duration, then a Start task button.
- R-13 `tab` cycles focus project → name → duration → button → project; `↑`/`↓` move
  focus. `enter` starts the task from any focus. While the project suggestions (R-26) are
  open, `tab`, `↑`/`↓`, `enter`, and `esc` work them first.
- R-14 While a text input has focus, printable keys type into it. `ctrl+s`, `ctrl+o`,
  and `ctrl+c` (R-06) work from every focus; `esc` moves focus to the button. The key
  hints change with focus.
- R-15 Task name is required, trimmed, and at most 100 characters. Starting with an empty
  name or invalid duration shows an inline error and does not start.
- R-16 Starting creates the task and opens the Active screen.

### Projects

- R-25 A task has an optional project: trimmed, at most 50 characters, blank for none. A
  project matching an existing one case-insensitively uses the existing project and its
  stored spelling (D-15). Projects are not renamed or deleted from the UI.
- R-26 While the Project input has focus, a dropdown suggests up to 5 existing projects:
  the most recently used when the input is empty, otherwise those containing the typed
  text (case-insensitive), prefix matches first, then by most recent use. The top match
  shows as dimmed ghost text when it extends the typed text. `↑`/`↓` move a highlight;
  `tab` fills in the highlighted (or else the top) suggestion when it differs from the
  input, and otherwise moves focus (with the input empty and nothing highlighted, `tab`
  moves focus); `enter` on a highlighted suggestion fills it in;
  `esc` closes the dropdown, and a second `esc` moves focus to the button.

### Active screen

- R-17 Shows a status badge (D-09: RUNNING, ENDING, OVERTIME, or PAUSED); beneath it the
  project in bold and the task name in regular dimmed text (with no project, the task
  name alone in bold); the remaining time in large block digits as `H:MM:SS` (or `MM:SS`
  under an hour; plain bold text when the terminal is too narrow); a progress bar of
  elapsed / estimate with the elapsed percentage; and beneath it the elapsed time and the
  estimate as text, `Elapsed 12:31 · Estimate 25:00`, or once time has been added the
  original estimate and the total added, `Elapsed 32:31 · Estimate 25:00 + 10:00 added`.
  The remaining time is not repeated as text.
- R-18 The timer and bar change colour by remaining fraction of the estimate (D-09):
  neutral, then info, then warning, then error. Overtime shows `+MM:SS` counting up in
  the error colour with a full bar.
- R-19 The display updates at least once per second.
- R-20 `p` pauses; while paused the timer is frozen, the screen shows PAUSED, and `p`
  resumes. Pauses are stored with start and end times.
- R-21 `d` opens a duration input (same grammar as R-10). `enter` adds it to the
  estimate; `esc` closes it unchanged. Works while running or paused.
- R-22 `n` opens a cancel confirmation. Confirming (`y` or `enter` on Yes) marks the task
  cancelled and opens the Start screen; `n`/`esc` returns unchanged.
- R-23 `e` marks the task completed, shows a random message from a built-in list of at
  least 20 motivational messages, and on any key opens the Start screen.
- R-24 `ctrl+s` opens the Stats screen (from Start, or from Active when no input,
  confirmation, or alert is open); the timer keeps running.

### Time-up alert

- R-30 When remaining reaches zero on a running task, the app shows a modal alert over
  whichever screen is open (Active or Stats), sends one system notification, and starts
  a looping alarm sound.
- R-31 The alert offers Extend and Mark as completed (`←`/`→`/`tab` to choose, `enter`
  to select; `d` and `e` also work). The sound stops only when one is chosen or the app
  quits. Other screen keys are inactive while the alert is open, except `ctrl+c`.
- R-32 Extend from the alert opens the duration input; the entered amount becomes the new
  remaining time, counted from now (D-10). Escaping the input returns to the alert.
- R-33 Notifications use `osascript` on macOS and `notify-send` on Ubuntu (D-13). If the
  tool is missing or fails, the app carries on and logs it.
- R-34 The sound is the one chosen on the Sound screen (R-51), synthesised by the app
  (D-13, D-16), looped with `afplay` (macOS) or `paplay`, then `aplay` (Ubuntu). With no
  player available, the app rings the terminal bell every 2 seconds instead.

### Alarm sound

- R-51 The Start screen shows the saved sound beside the Start task button as
  `Alarm: <name>`. `ctrl+o` on the Start screen, from any focus, opens the Sound screen:
  Chime, Bell, Beep, Marimba, and Rising, with the saved choice marked and highlighted.
  The default is Chime (D-16).
- R-52 `↑`/`↓` move the highlight and play the highlighted sound once, stopping any
  sound still previewing. `enter` saves the highlighted sound and returns to Start; `esc`
  returns without changing it. Leaving the screen stops the preview.
- R-53 The choice is stored in the database, survives restarts, and is what the next
  alarm plays. A missing or unknown stored value means Chime. If saving fails, the
  screen shows the error and keeps the previous choice.

### Stats screen

- R-40 Periods: 7 days (default each time the screen opens), 30 days, 3 months, as defined
  in D-11. `tab` cycles them.
- R-41 Scorecard: number of completed tasks in the period, with the previous period's
  number and the change as `+n`/`-n` and a percentage (`—` when the previous is 0).
- R-42 Scorecard: average completed tasks per day in the period (one decimal), with the
  same comparison.
- R-43 A vertical bar chart with an estimated and an actual bar per bucket, a legend, and
  a y-axis in hours and minutes. Buckets and x-axis labels (moment.js tokens): 7 days =
  7 daily buckets, `ddd MMM DD`; 30 days = 7-day buckets, `MMM DD` of the bucket start;
  3 months = 3 calendar months, `MMM YYYY`.
- R-44 Only completed tasks count in stats, attributed to the local date they were
  completed. Cancelled and active tasks are excluded.
- R-45 With no completed tasks in the period, the chart area shows an empty-state message.
- R-46 `esc` returns to the previous screen in the state it was left.
- R-47 `e` exports all data (R-50) and shows the written path or the error in the screen.

### Export

- R-50 Export writes every task (all statuses) with its pauses and extensions to
  `unitock-export-YYYYMMDD-HHMMSS.json` (local time) in the current working directory, in
  the format under Interfaces. It never overwrites an existing file.

## Interfaces

These are the public interface protected by Semantic Versioning.

**Command line**

```
unitock [--data-dir DIR] [--version]
```

- `--data-dir` sets the data directory. Default: `$XDG_DATA_HOME/unitock`, or
  `~/.local/share/unitock` when `XDG_DATA_HOME` is unset, on both macOS and Ubuntu.
  Created with mode 0700 if missing.
- `--version` prints `unitock <version> (<commit>, <date>)` and exits 0.
- Exit codes: 0 normal quit; 1 startup error (bad flag, data dir or database unusable,
  another instance running).

**Data directory:** `unitock.db` (SQLite, schema per D-07, migrated on startup),
`unitock.lock` (instance lock), `unitock.log` (diagnostic log), `alarm.wav` (the
chosen alarm sound, rewritten when it changes, D-13, D-16). The database schema is
internal; the export format is the stable way to read data.

**Export JSON, schema version 1** (D-14). Times are RFC 3339 with offset; durations are
whole seconds.

```json
{
  "schema_version": 1,
  "app_version": "1.0.0",
  "exported_at": "2026-11-17T14:03:00-08:00",
  "tasks": [
    {
      "id": 1,
      "name": "Write spec",
      "project": "unitock",
      "status": "completed",
      "started_at": "2026-11-17T13:00:00-08:00",
      "ended_at": "2026-11-17T13:52:10-08:00",
      "original_estimate_seconds": 2700,
      "estimate_seconds": 3600,
      "actual_seconds": 2950,
      "paused_seconds": 180,
      "extensions": [{ "at": "2026-11-17T13:45:00-08:00", "seconds": 900 }],
      "pauses": [{ "started_at": "2026-11-17T13:20:00-08:00",
                   "ended_at": "2026-11-17T13:23:00-08:00" }]
    }
  ]
}
```

`project` is null for a task without one; `ended_at` is null for an active task; `actual_seconds` is elapsed at export time.

## Constraints

- Go 1.26 or newer. Static binaries with `CGO_ENABLED=0`, so the SQLite driver is pure Go
  (D-05).
- Platforms: macOS 14+ (arm64, amd64) and Ubuntu 22.04+ (amd64, arm64). Released as
  GitHub Releases binaries and installable with `go install` (D-06).
- Terminals: works in 256-colour and true-colour terminals; respects `NO_COLOR`.
- No network access at runtime. External programs are limited to `osascript`,
  `notify-send`, `afplay`, `paplay`, and `aplay`, called with fixed arguments, never
  through a shell.
- Startup to first frame under 300 ms and stats render under 100 ms with 10,000 tasks.
- Core packages (duration parsing, timer maths, stats) take the clock as a parameter so
  tests need no real time or sleeps.

## End-to-end verification

On each of macOS and Ubuntu, with `--data-dir` pointing at a scratch directory:

1. Launch; type `Write spec`, tab, `1m`, enter. Active screen shows ~`01:00` counting down.
2. Press `p`, wait 10 s, press `p`: the timer lost no time while paused.
3. Press `d`, enter `1m`: the row reads `Estimate 01:00 + 01:00 added` and remaining
   grows by a minute.
4. Press `ctrl+c`, wait 20 s, relaunch: the task resumes with 20 s less remaining.
5. Press `ctrl+s` and wait for zero on the Stats screen: the alert appears over Stats, a
   system notification shows, and the chime loops. Choose Extend `1m`: sound stops,
   remaining is 1:00.
6. Let it run out again; choose Mark as completed: a motivational message shows, then the
   Start screen.
7. Start and cancel a second task with `n`, `y`.
8. Open Stats: 1 completed task today in each period; the chart shows its estimated and
   actual time; the cancelled task is absent. `tab` cycles 30 days and 3 months with the
   right labels.
9. Press `e`: a JSON file appears in the working directory with both tasks, pauses, and
   extensions, matching the export format.
10. Start a second instance with the same `--data-dir`: it exits 1 naming the lock file.
11. In the first instance, on Start the button shows `Alarm: Chime`. While typing in
    the Task name input, press `ctrl+o`: the Sound screen opens with Chime marked.
    Press `↓` through the list: each sound plays once and they are clearly different.
    Choose Bell with `enter`: Start shows `Alarm: Bell`. Quit, relaunch, start a `10s`
    task, and wait: the alert loops Bell. Complete it. Pressing `q` or `s` anywhere
    never quits or opens Stats.

## Open questions

RFI-02 (repository rename to `unitock`).
