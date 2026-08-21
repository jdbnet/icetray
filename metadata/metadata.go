package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NowPlaying holds track metadata from an Icecast server when available.
type NowPlaying struct {
	Station   string `json:"station"`
	Title     string `json:"title"`
	Genre     string `json:"genre,omitempty"`
	Listeners int    `json:"listeners,omitempty"`
}

type statusJSON struct {
	Icestats struct {
		Source []struct {
			Listeners   int    `json:"listeners"`
			ServerName  string `json:"server_name"`
			ServerType  string `json:"server_type"`
			Genre       string `json:"genre"`
			Title       string `json:"title"`
			ListenURL   string `json:"listenurl"`
			ServerURL   string `json:"server_url"`
			StreamStart string `json:"stream_start"`
		} `json:"source"`
	} `json:"icestats"`
}

// Fetch polls the Icecast status-json endpoint for the given stream URL.
func Fetch(ctx context.Context, streamURL string) (NowPlaying, error) {
	base, mount, err := parseStreamURL(streamURL)
	if err != nil {
		return NowPlaying{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/status-json.xsl", nil)
	if err != nil {
		return NowPlaying{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return NowPlaying{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NowPlaying{}, fmt.Errorf("status-json returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return NowPlaying{}, err
	}

	var data statusJSON
	if err := json.Unmarshal(body, &data); err != nil {
		return NowPlaying{}, err
	}

	sources := data.Icestats.Source
	if len(sources) == 0 {
		return NowPlaying{}, fmt.Errorf("no sources in status-json")
	}

	normalizedMount := normalizeMount(mount)
	for _, src := range sources {
		if sourceMatches(src.ListenURL, src.ServerURL, normalizedMount) {
			np := NowPlaying{
				Station:   src.ServerName,
				Title:     src.Title,
				Genre:     src.Genre,
				Listeners: src.Listeners,
			}
			if np.Station == "" {
				np.Station = src.ServerType
			}
			return np, nil
		}
	}

	// Fallback to first source if only one mount
	if len(sources) == 1 {
		src := sources[0]
		return NowPlaying{
			Station:   src.ServerName,
			Title:     src.Title,
			Genre:     src.Genre,
			Listeners: src.Listeners,
		}, nil
	}

	return NowPlaying{}, fmt.Errorf("mount not found in status-json")
}

func parseStreamURL(streamURL string) (base string, mount string, err error) {
	u, err := url.Parse(streamURL)
	if err != nil {
		return "", "", err
	}
	mount = u.Path
	if mount == "" {
		mount = "/"
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	base = strings.TrimRight(u.String(), "/")
	return base, mount, nil
}

func normalizeMount(mount string) string {
	mount = strings.TrimSpace(mount)
	if mount == "" {
		return "/"
	}
	if !strings.HasPrefix(mount, "/") {
		mount = "/" + mount
	}
	return mount
}

func sourceMatches(listenURL, serverURL, mount string) bool {
	for _, raw := range []string{listenURL, serverURL} {
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if normalizeMount(u.Path) == mount {
			return true
		}
		if strings.HasSuffix(strings.ToLower(u.Path), strings.ToLower(mount)) {
			return true
		}
	}
	return false
}

// Poller periodically fetches metadata and invokes the callback.
type Poller struct {
	interval time.Duration
	onUpdate func(NowPlaying)
}

// NewPoller creates a metadata poller.
func NewPoller(interval time.Duration, onUpdate func(NowPlaying)) *Poller {
	return &Poller{
		interval: interval,
		onUpdate: onUpdate,
	}
}
func (p *Poller) Run(ctx context.Context, streamURL string) {
	if streamURL == "" {
		return
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.poll(ctx, streamURL)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx, streamURL)
		}
	}
}

func (p *Poller) poll(ctx context.Context, streamURL string) {
	np, err := Fetch(ctx, streamURL)
	if err != nil {
		return
	}
	if p.onUpdate != nil {
		p.onUpdate(np)
	}
}
