package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

type sourceEntry struct {
	Mount string
	Data  map[string]any
}

// Fetch polls Icecast for now playing metadata using publicstats, legacy status-json, and ICY fallbacks.
func Fetch(ctx context.Context, streamURL string) (NowPlaying, error) {
	base, mount, err := parseStreamURL(streamURL)
	if err != nil {
		return NowPlaying{}, err
	}

	if np, err := fetchPublicStats(ctx, base, mount); err == nil {
		return np, nil
	}

	if np, err := fetchLegacyStats(ctx, base, mount); err == nil {
		return np, nil
	}

	if np, err := fetchICYMetadata(ctx, streamURL); err == nil {
		return np, nil
	}

	return NowPlaying{}, errors.New("no metadata available")
}

func fetchPublicStats(ctx context.Context, base, mount string) (NowPlaying, error) {
	body, err := fetchURL(ctx, base+"/admin/publicstats.json")
	if err != nil {
		return NowPlaying{}, err
	}

	var docs []json.RawMessage
	if err := json.Unmarshal(body, &docs); err != nil {
		return NowPlaying{}, err
	}

	for _, doc := range docs {
		var block struct {
			Source json.RawMessage `json:"source"`
		}
		if err := json.Unmarshal(doc, &block); err != nil || len(block.Source) == 0 {
			continue
		}

		entries, err := parseSourceEntries(block.Source)
		if err != nil {
			continue
		}

		if np, ok := matchSource(entries, mount); ok {
			return np, nil
		}
	}

	return NowPlaying{}, errors.New("mount not found in publicstats")
}

func fetchLegacyStats(ctx context.Context, base, mount string) (NowPlaying, error) {
	body, err := fetchURL(ctx, base+"/status-json.xsl")
	if err != nil {
		return NowPlaying{}, err
	}

	var data struct {
		Icestats struct {
			Source json.RawMessage `json:"source"`
		} `json:"icestats"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return NowPlaying{}, err
	}

	entries, err := parseSourceEntries(data.Icestats.Source)
	if err != nil {
		return NowPlaying{}, err
	}
	if len(entries) == 0 {
		return NowPlaying{}, errors.New("no sources in status-json")
	}

	if np, ok := matchSource(entries, mount); ok {
		return np, nil
	}

	return NowPlaying{}, errors.New("mount not found in status-json")
}

func fetchICYMetadata(ctx context.Context, streamURL string) (NowPlaying, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return NowPlaying{}, err
	}
	req.Header.Set("Icy-MetaData", "1")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return NowPlaying{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NowPlaying{}, fmt.Errorf("stream returned %d", resp.StatusCode)
	}

	np := NowPlaying{
		Station: firstHeader(resp.Header, "Icy-Name", "Ice-Name", "icy-name"),
		Genre:   firstHeader(resp.Header, "Icy-Genre", "Ice-Genre", "icy-genre"),
	}

	metaInt := firstHeader(resp.Header, "Icy-MetaInt", "icy-metaint")
	if metaInt == "" {
		if np.Title != "" || np.Station != "" {
			return np, nil
		}
		return NowPlaying{}, errors.New("stream has no icy-metaint")
	}

	blockSize, err := strconv.Atoi(metaInt)
	if err != nil || blockSize <= 0 {
		return NowPlaying{}, errors.New("invalid icy-metaint")
	}

	title, err := readFirstICYTitle(resp.Body, blockSize)
	if err == nil && title != "" {
		np.Title = title
	}

	if np.Title == "" && np.Station == "" {
		return NowPlaying{}, errors.New("no icy metadata")
	}

	return np, nil
}

func readFirstICYTitle(r io.Reader, blockSize int) (string, error) {
	audio := make([]byte, blockSize)
	if _, err := io.ReadFull(r, audio); err != nil {
		return "", err
	}

	var metaLenByte [1]byte
	if _, err := io.ReadFull(r, metaLenByte[:]); err != nil {
		return "", err
	}

	metaLen := int(metaLenByte[0]) * 16
	if metaLen == 0 {
		return "", errors.New("empty icy metadata block")
	}

	meta := make([]byte, metaLen)
	if _, err := io.ReadFull(r, meta); err != nil {
		return "", err
	}

	return parseStreamTitle(string(meta)), nil
}

func parseStreamTitle(meta string) string {
	for _, part := range strings.Split(meta, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), "StreamTitle") {
			return strings.Trim(val, " '\"") 
		}
	}
	return ""
}

func parseSourceEntries(raw json.RawMessage) ([]sourceEntry, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, errors.New("empty source")
	}

	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil {
		entries := make([]sourceEntry, 0, len(arr))
		for _, item := range arr {
			entries = append(entries, sourceEntry{Data: item})
		}
		return entries, nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}

	if listenURL, ok := obj["listenurl"]; ok {
		var single map[string]any
		if err := json.Unmarshal(raw, &single); err != nil {
			return nil, err
		}
		_ = listenURL
		return []sourceEntry{{Data: single}}, nil
	}

	entries := make([]sourceEntry, 0, len(obj))
	for mount, itemRaw := range obj {
		var item map[string]any
		if err := json.Unmarshal(itemRaw, &item); err != nil {
			continue
		}
		entries = append(entries, sourceEntry{Mount: mount, Data: item})
	}
	if len(entries) == 0 {
		return nil, errors.New("no parseable sources")
	}
	return entries, nil
}

func matchSource(entries []sourceEntry, mount string) (NowPlaying, bool) {
	normalizedMount := normalizeMount(mount)

	var fallback NowPlaying
	var hasFallback bool

	for _, entry := range entries {
		np := entryToNowPlaying(entry.Data)
		if np.Station == "" && np.Title == "" && np.Genre == "" && np.Listeners == 0 {
			continue
		}

		if entry.Mount != "" && mountsEqual(entry.Mount, normalizedMount) {
			return np, true
		}

		listenURL := stringField(entry.Data, "listenurl", "ListenURL")
		if listenURL != "" && sourceMatches(listenURL, "", normalizedMount) {
			return np, true
		}

		if !hasFallback {
			fallback = np
			hasFallback = true
		}
	}

	if len(entries) == 1 && hasFallback {
		return fallback, true
	}

	return NowPlaying{}, false
}

func entryToNowPlaying(data map[string]any) NowPlaying {
	np := NowPlaying{
		Station:   firstStringField(data, "server_name", "server_description", "ServerName"),
		Title:     extractTitle(data),
		Genre:     stringField(data, "genre", "Genre"),
		Listeners: intField(data, "listeners", "Listeners"),
	}
	if np.Station == "" {
		np.Station = stringField(data, "server_type", "ServerType")
	}
	return np
}

func extractTitle(data map[string]any) string {
	for _, key := range []string{"display-title", "display_title", "title", "Title"} {
		if value := stringField(data, key); value != "" {
			return value
		}
	}

	if meta, ok := data["metadata"].(map[string]any); ok {
		for _, key := range []string{"streamtitle", "StreamTitle", "x_icy_title", "icy-title", "title", "Title"} {
			if value := stringField(meta, key); value != "" {
				return value
			}
		}
	}

	if playlist, ok := data["playlist"].(map[string]any); ok {
		if title := lastPlaylistTitle(playlist); title != "" {
			return title
		}
	}

	return ""
}

func lastPlaylistTitle(playlist map[string]any) string {
	inner, ok := playlist["playlist"].(map[string]any)
	if !ok {
		return ""
	}

	tracks, ok := inner["track"].([]any)
	if !ok || len(tracks) == 0 {
		return ""
	}

	last, ok := tracks[len(tracks)-1].(map[string]any)
	if !ok {
		return ""
	}

	return stringField(last, "title", "Title")
}

func stringField(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			switch typed := value.(type) {
			case string:
				if typed != "" {
					return typed
				}
			case json.Number:
				return typed.String()
			}
		}
	}
	return ""
}

func firstStringField(data map[string]any, keys ...string) string {
	return stringField(data, keys...)
}

func intField(data map[string]any, keys ...string) int {
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int(typed)
		case int:
			return typed
		case json.Number:
			i, _ := typed.Int64()
			return int(i)
		case string:
			i, err := strconv.Atoi(typed)
			if err == nil {
				return i
			}
		}
	}
	return 0
}

func firstHeader(header http.Header, keys ...string) string {
	for _, key := range keys {
		if value := header.Get(key); value != "" {
			return value
		}
	}
	return ""
}

func fetchURL(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d", endpoint, resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
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

func mountsEqual(a, b string) bool {
	return normalizeMount(a) == normalizeMount(b)
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

// Run polls until the context is cancelled.
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
