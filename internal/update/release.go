package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LatestReleaseURL ist die Auskunft von GitHub über das neueste Release.
// Vorabversionen und Entwürfe liefert sie nicht.
const LatestReleaseURL = "https://api.github.com/repos/mxzinke/buchfink/releases/latest"

// checksumsName ist die Datei mit den SHA-256-Summen aller Dateien eines
// Release (scripts/release.sh).
const checksumsName = "SHA256SUMS"

const (
	maxMetadataBytes = 1 << 20
	// maxAssetBytes begrenzt den Download: die größte Datei eines Release
	// liegt bei 70 MB.
	maxAssetBytes = 512 << 20
)

// Release ist eine veröffentlichte Fassung.
type Release struct {
	Version     string `json:"version"`
	PublishedAt string `json:"publishedAt"`
	// Notes ist der Eintrag der Änderungshistorie als Markdown.
	Notes   string `json:"notes"`
	PageURL string `json:"pageUrl"`

	asset     asset
	checksums string
}

type asset struct {
	name string
	url  string
	size int64
}

// assetName ist die Datei eines Release für ein Betriebssystem, leer, wenn es
// keine gibt. Die Namen vergibt scripts/release.sh.
func assetName(goos, goarch string) string {
	switch {
	case goos == "darwin":
		return "buchfink-darwin-universal.zip"
	case goos == "windows" && goarch == "amd64":
		return "buchfink-windows-amd64.exe"
	case goos == "linux" && goarch == "amd64":
		return "buchfink-linux-amd64.tar.gz"
	}
	return ""
}

// allowedHost begrenzt jeden Abruf auf GitHub: die Auskunft, die Seite des
// Release und die Dateiablage, auf die GitHub Downloads umleitet.
func allowedHost(host string) bool {
	host = strings.ToLower(host)
	return host == "github.com" || host == "api.github.com" || strings.HasSuffix(host, ".githubusercontent.com")
}

func newClient() *http.Client {
	return &http.Client{
		// Die Frist gilt dem Download einer Programmdatei über eine langsame
		// Leitung; die Suche begrenzt ihr Aufrufer.
		Timeout: 30 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("zu viele Umleitungen")
			}
			if req.URL.Scheme != "https" || !allowedHost(req.URL.Hostname()) {
				return fmt.Errorf("Umleitung auf %s ist nicht zugelassen", req.URL.Host)
			}
			return nil
		},
	}
}

func (s *Service) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Buchfink/"+s.current)
	req.Header.Set("Accept", "application/vnd.github+json, application/octet-stream")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s antwortet mit %s", req.URL.Host, resp.Status)
	}
	return resp, nil
}

// latest fragt das neueste Release ab und sucht die Datei für dieses System.
func (s *Service) latest(ctx context.Context) (*Release, error) {
	resp, err := s.get(ctx, s.endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload struct {
		TagName     string `json:"tag_name"`
		Body        string `json:"body"`
		HTMLURL     string `json:"html_url"`
		PublishedAt string `json:"published_at"`
		Draft       bool   `json:"draft"`
		Prerelease  bool   `json:"prerelease"`
		Assets      []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxMetadataBytes)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("die Auskunft über das neueste Release ist nicht lesbar: %w", err)
	}
	if payload.Draft || payload.Prerelease {
		return nil, fmt.Errorf("die Auskunft nennt kein veröffentlichtes Release")
	}
	if _, ok := parseVersion(payload.TagName); !ok {
		return nil, fmt.Errorf("das neueste Release trägt keine Fassung: %q", payload.TagName)
	}
	release := &Release{
		Version:     strings.TrimPrefix(payload.TagName, "v"),
		PublishedAt: payload.PublishedAt,
		Notes:       strings.TrimSpace(payload.Body),
		PageURL:     payload.HTMLURL,
	}
	want := assetName(s.goos, s.goarch)
	for _, a := range payload.Assets {
		switch a.Name {
		case checksumsName:
			release.checksums = a.URL
		case want:
			release.asset = asset{name: a.Name, url: a.URL, size: a.Size}
		}
	}
	return release, nil
}

// expectedSum liest aus SHA256SUMS die Summe der Datei name.
func (s *Service) expectedSum(ctx context.Context, release *Release, name string) (string, error) {
	if release.checksums == "" {
		return "", fmt.Errorf("das Release enthält keine Prüfsummen (%s)", checksumsName)
	}
	resp, err := s.get(ctx, release.checksums)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMetadataBytes))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		// shasum schreibt „<summe>  <datei>", im Binärmodus „<summe> *<datei>".
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != name {
			continue
		}
		sum := strings.ToLower(fields[0])
		if raw, err := hex.DecodeString(sum); err != nil || len(raw) != sha256.Size {
			return "", fmt.Errorf("die Prüfsumme für %s ist nicht lesbar", name)
		}
		return sum, nil
	}
	return "", fmt.Errorf("für %s steht keine Prüfsumme im Release", name)
}
