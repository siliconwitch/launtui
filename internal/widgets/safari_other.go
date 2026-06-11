//go:build !darwin

package widgets

func safariSupported() bool { return false }

func loadSafariEntries(cfg SafariConfig) ([]safariEntry, bool) { return nil, false }

func activateSafariEntry(entry safariEntry) {}
