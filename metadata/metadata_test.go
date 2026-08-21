package metadata

import (
	"context"
	"encoding/json"
	"testing"
)

func TestParseSourceEntriesSingleObject(t *testing.T) {
	raw := json.RawMessage(`{
		"listeners": 11,
		"listenurl": "http://streaming.eguzki.eus:8000/eguzki.mp3",
		"server_name": "Eguzki Irratia",
		"genre": "denetarik"
	}`)

	entries, err := parseSourceEntries(raw)
	if err != nil {
		t.Fatalf("parseSourceEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Data["server_name"] != "Eguzki Irratia" {
		t.Fatalf("unexpected server_name: %v", entries[0].Data["server_name"])
	}
}

func TestParseSourceEntriesMountMap(t *testing.T) {
	raw := json.RawMessage(`{
		"/eguzki.mp3": {
			"display-title": "Artist - Song",
			"listeners": 3,
			"listenurl": "http://streaming.eguzki.eus:8000/eguzki.mp3",
			"server_name": "Eguzki Irratia"
		}
	}`)

	entries, err := parseSourceEntries(raw)
	if err != nil {
		t.Fatalf("parseSourceEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Mount != "/eguzki.mp3" {
		t.Fatalf("mount = %q", entries[0].Mount)
	}
}

func TestMatchSourceDisplayTitle(t *testing.T) {
	entries := []sourceEntry{{
		Mount: "/stream.mp3",
		Data: map[string]any{
			"display-title": "Current Track",
			"server_name":   "My Station",
			"listeners":     float64(5),
			"listenurl":     "https://example.com/stream.mp3",
		},
	}}

	np, ok := matchSource(entries, "/stream.mp3")
	if !ok {
		t.Fatal("matchSource() = false, want true")
	}
	if np.Title != "Current Track" {
		t.Fatalf("Title = %q", np.Title)
	}
	if np.Station != "My Station" {
		t.Fatalf("Station = %q", np.Station)
	}
	if np.Listeners != 5 {
		t.Fatalf("Listeners = %d", np.Listeners)
	}
}

func TestExtractTitleFromMetadata(t *testing.T) {
	data := map[string]any{
		"metadata": map[string]any{
			"streamtitle": "Artist - Track Name",
		},
	}
	if got := extractTitle(data); got != "Artist - Track Name" {
		t.Fatalf("extractTitle() = %q", got)
	}
}

func TestParseStreamTitle(t *testing.T) {
	meta := "StreamTitle='Artist - Song';StreamUrl='http://example.com';"
	if got := parseStreamTitle(meta); got != "Artist - Song" {
		t.Fatalf("parseStreamTitle() = %q", got)
	}
}

func TestFetchLegacyStatsIcecast25Sample(t *testing.T) {
	body := []byte(`{"icestats":{"source":{"listeners":11,"listenurl":"http://streaming.eguzki.eus:8000/eguzki.mp3","server_name":"Eguzki Irratia","genre":"denetarik"}}}`)

	var data struct {
		Icestats struct {
			Source json.RawMessage `json:"source"`
		} `json:"icestats"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	entries, err := parseSourceEntries(data.Icestats.Source)
	if err != nil {
		t.Fatalf("parseSourceEntries() error = %v", err)
	}

	np, ok := matchSource(entries, "/eguzki.mp3")
	if !ok {
		t.Fatal("matchSource() = false, want true")
	}
	if np.Station != "Eguzki Irratia" {
		t.Fatalf("Station = %q", np.Station)
	}
}

func TestFetchPublicStatsSample(t *testing.T) {
	body := []byte(`[{"name":"icestats","ns":"http://icecast.org/specs/legacystats-0.0.1"},{"source":{"/eguzki.mp3":{"display-title":"Live Show","genre":"denetarik","listeners":11,"listenurl":"http://streaming.eguzki.eus:8000/eguzki.mp3","server_name":"Eguzki Irratia"}}}]`)

	var docs []json.RawMessage
	if err := json.Unmarshal(body, &docs); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	var block struct {
		Source json.RawMessage `json:"source"`
	}
	if err := json.Unmarshal(docs[1], &block); err != nil {
		t.Fatalf("json.Unmarshal(block) error = %v", err)
	}

	entries, err := parseSourceEntries(block.Source)
	if err != nil {
		t.Fatalf("parseSourceEntries() error = %v", err)
	}

	np, ok := matchSource(entries, "/eguzki.mp3")
	if !ok {
		t.Fatal("matchSource() = false, want true")
	}
	if np.Title != "Live Show" {
		t.Fatalf("Title = %q", np.Title)
	}
}

func TestFetchIntegrationEguzkiPublicStats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	np, err := Fetch(context.Background(), "http://streaming.eguzki.eus:8000/eguzki.mp3")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if np.Station == "" {
		t.Fatal("expected station name from publicstats")
	}
}
