package widgets

import "testing"

func TestRunExcludesApps(t *testing.T) {
	cfg := DefaultRunConfig()
	cfg.Exclude = []string{"firefox", "  Slack  "}

	run := NewRun(cfg)

	updated, _ := run.Update(appsLoadedMsg{
		{Name: "Firefox", Exec: "firefox"},
		{Name: "Slack", Exec: "slack"},
		{Name: "Terminal", Exec: "term"},
	})

	apps := updated.(Run).apps

	if len(apps) != 1 || apps[0].Name != "Terminal" {
		t.Fatalf("apps = %+v, want only Terminal", apps)
	}
}

func TestRunCursorResetsOnQueryChange(t *testing.T) {
	mode, _ := NewRun(DefaultRunConfig()).Update(appsLoadedMsg{
		{Name: "alpha", Exec: "a"},
		{Name: "beta", Exec: "b"},
		{Name: "gamma", Exec: "c"},
	})

	moved := mode.MoveDown().MoveDown().(Run)

	if moved.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", moved.cursor)
	}

	typed := moved.SetQuery("a").(Run)

	if typed.cursor != 0 {
		t.Fatalf("cursor after typing = %d, want 0", typed.cursor)
	}
}
