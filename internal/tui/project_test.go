package tui

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
)

// seedProjects stores one finished task per project, oldest first, so the last name is
// the most recently used, then reloads the Start screen's suggestions.
func (h *harness) seedProjects(names ...string) {
	h.t.Helper()
	ctx := context.Background()
	at := t0.Add(-time.Duration(len(names)+1) * time.Hour)
	for _, p := range names {
		tk, err := h.store.Create(ctx, p, "seed", time.Hour, at)
		if err != nil {
			h.t.Fatal(err)
		}
		if err := h.store.Complete(ctx, tk.ID, at.Add(time.Minute)); err != nil {
			h.t.Fatal(err)
		}
		at = at.Add(time.Hour)
	}
	h.run(loadProjects(h.m.deps))
}

func TestProjectDropdownListsRecentWhenEmpty(t *testing.T) {
	h := newHarness(t)
	h.seedProjects("a1", "a2", "a3", "a4", "a5", "a6")
	if got, want := h.m.start.suggestions(), []string{"a6", "a5", "a4", "a3"}; !slices.Equal(got, want) {
		t.Errorf("suggestions = %v, want %v (4 most recent)", got, want)
	}
	if !h.m.start.dropdownOpen() || !strings.Contains(h.view(), "a6") || strings.Contains(h.view(), "a2") {
		t.Errorf("dropdown view:\n%s", h.view())
	}
	if !strings.Contains(h.view(), "tab complete") {
		t.Errorf("hints do not describe the dropdown:\n%s", h.view())
	}
	// Tab on an empty input skips the project instead of filling one in.
	h.press("tab")
	if h.m.start.focus != focusName || h.m.start.project.Value() != "" {
		t.Errorf("tab on empty project: focus %d value %q", h.m.start.focus, h.m.start.project.Value())
	}
	if h.m.start.dropdownOpen() || strings.Contains(h.view(), "a6") {
		t.Error("dropdown still shown after focus left the project input")
	}
}

func TestProjectTabCompletesThenMovesFocus(t *testing.T) {
	h := newHarness(t)
	h.seedProjects("unitock", "untitled")
	h.typeText("UNI")
	if got := h.m.start.suggestions(); !slices.Equal(got, []string{"unitock"}) {
		t.Fatalf("suggestions = %v", got)
	}
	if v := h.view(); !strings.Contains(v, "tock") { // ghost text after the typed "UNI"
		t.Errorf("no ghost completion in view:\n%s", v)
	}
	h.press("tab")
	if got := h.m.start.project.Value(); got != "unitock" || h.m.start.focus != focusProject {
		t.Fatalf("after tab: value %q focus %d; want stored spelling, still on project", got, h.m.start.focus)
	}
	h.press("tab")
	if h.m.start.focus != focusName {
		t.Errorf("second tab: focus %d, want name", h.m.start.focus)
	}
}

func TestProjectSuggestionsPrefixFirstThenContains(t *testing.T) {
	h := newHarness(t)
	h.seedProjects("tockpit", "unitock", "tock tools", "other")
	h.typeText("tock")
	want := []string{"tock tools", "tockpit", "unitock"} // prefixes by recency, then contains
	if got := h.m.start.suggestions(); !slices.Equal(got, want) {
		t.Errorf("suggestions = %v, want %v", got, want)
	}
}

func TestProjectArrowsHighlightAndEnterChooses(t *testing.T) {
	h := newHarness(t)
	h.seedProjects("beta", "alpha") // alpha most recent
	h.press("down", "down")
	if h.m.start.pick != 1 || h.m.start.focus != focusProject {
		t.Fatalf("pick %d focus %d; arrows must work the open dropdown", h.m.start.pick, h.m.start.focus)
	}
	h.press("down") // stays on the last row
	h.press("up", "up", "up")
	if h.m.start.pick != -1 {
		t.Fatalf("pick %d after moving up past the top, want none", h.m.start.pick)
	}
	h.press("down", "down")
	if !strings.Contains(h.view(), "enter choose") {
		t.Errorf("hints do not show enter choose:\n%s", h.view())
	}
	h.press("enter")
	if got := h.m.start.project.Value(); got != "beta" || h.m.screen != screenStart {
		t.Errorf("enter on highlight: value %q screen %d, want beta on Start", got, h.m.screen)
	}
	// Highlight then tab takes the highlighted row, not the top one.
	h.m.start.project.SetValue("")
	h.press("down", "down", "tab")
	if got := h.m.start.project.Value(); got != "beta" {
		t.Errorf("tab on highlight: value %q, want beta", got)
	}
}

func TestProjectEscClosesThenFocusesButton(t *testing.T) {
	h := newHarness(t)
	h.seedProjects("unitock")
	h.typeText("u")
	h.press("esc")
	if h.m.start.dropdownOpen() || h.m.start.focus != focusProject {
		t.Fatalf("first esc: open %v focus %d; want closed, still on project", h.m.start.dropdownOpen(), h.m.start.focus)
	}
	h.press("tab") // closed: tab moves focus without completing
	if h.m.start.focus != focusName || h.m.start.project.Value() != "u" {
		t.Fatalf("tab with closed dropdown: focus %d value %q", h.m.start.focus, h.m.start.project.Value())
	}
	h.press("shift+tab")
	h.typeText("n") // typing reopens
	if !h.m.start.dropdownOpen() {
		t.Fatal("typing did not reopen the dropdown")
	}
	h.press("esc", "esc")
	if h.m.start.focus != focusButton {
		t.Errorf("second esc: focus %d, want button", h.m.start.focus)
	}
}

func TestProjectEnterWithoutHighlightStarts(t *testing.T) {
	h := newHarness(t)
	h.seedProjects("unitock")
	h.typeText("UNITOCK")
	h.press("tab", "tab") // complete to the stored spelling, then move on
	h.typeText("write")
	h.press("tab")
	h.typeText("25m")
	h.press("shift+tab", "shift+tab", "enter") // back on the project, dropdown open, nothing highlighted
	if h.m.screen != screenActive {
		t.Fatalf("enter did not start: err %q", h.m.start.err)
	}
	got := h.m.active.task
	if got.Project != "unitock" || got.Name != "write" {
		t.Errorf("started %+v", got)
	}
	if v := h.view(); !strings.Contains(v, "unitock") || !strings.Contains(v, "write") {
		t.Errorf("Active view lacks project or name:\n%s", v)
	}
}

func TestProjectNewIsCreatedAndOptional(t *testing.T) {
	h := newHarness(t)
	h.typeText("  Side quest  ")
	h.press("tab")
	h.typeText("write")
	h.press("tab")
	h.typeText("25m")
	h.press("enter")
	if h.m.screen != screenActive || h.m.active.task.Project != "Side quest" {
		t.Fatalf("project = %q, err %q", h.m.active.task.Project, h.m.start.err)
	}
	ps, err := h.store.Projects(context.Background())
	if err != nil || !slices.Equal(ps, []string{"Side quest"}) {
		t.Errorf("stored projects = %v, %v", ps, err)
	}
}

func TestProjectTooLong(t *testing.T) {
	h := newHarness(t)
	h.typeText(strings.Repeat("p", MaxProjectLen+1))
	h.press("tab")
	h.typeText("write")
	h.press("tab")
	h.typeText("25m")
	h.press("enter")
	if !strings.Contains(h.m.start.err, "limit is 50") || h.m.screen != screenStart || h.taskCount() != 0 {
		t.Errorf("err %q screen %d tasks %d", h.m.start.err, h.m.screen, h.taskCount())
	}
}
