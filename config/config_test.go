package config

import (
	"testing"
)

func TestReorderStreams(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	a, err := cfg.AddStream("A", "https://a.example/stream")
	if err != nil {
		t.Fatal(err)
	}
	b, err := cfg.AddStream("B", "https://b.example/stream")
	if err != nil {
		t.Fatal(err)
	}
	c, err := cfg.AddStream("C", "https://c.example/stream")
	if err != nil {
		t.Fatal(err)
	}

	if err := cfg.ReorderStreams([]string{c.ID, a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.GetStreams()
	if len(got) != 3 {
		t.Fatalf("expected 3 streams, got %d", len(got))
	}
	if got[0].ID != c.ID || got[1].ID != a.ID || got[2].ID != b.ID {
		t.Fatalf("unexpected order: %+v", got)
	}

	if err := cfg.ReorderStreams([]string{a.ID}); err == nil {
		t.Fatal("expected length mismatch error")
	}
}
