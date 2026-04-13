package cockpit_srv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	releasesCacheTTL  = 5 * time.Minute
	releasesMaxLimit  = 20
	releasesDefLimit  = 5
	releasesPerPage   = 20
	githubReleasesURL = "https://api.github.com/repos/%s/%s/releases?per_page=%d"
)

type githubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
}

type releasesCache struct {
	mu        sync.Mutex
	releases  []githubRelease
	fetchedAt time.Time
}

func (s *CockpitServer) releasesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse limit param
	limit := releasesDefLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
		if n > releasesMaxLimit {
			n = releasesMaxLimit
		}
		limit = n
	}

	releases, err := s.getCachedReleases()
	if err != nil {
		s.log.Error().Err(err).Msg("failed to fetch GitHub releases")
		http.Error(w, "Failed to fetch releases", http.StatusBadGateway)
		return
	}

	if limit < len(releases) {
		releases = releases[:limit]
	}

	md := formatReleasesMarkdown(releases)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	fmt.Fprint(w, md)
}

func (s *CockpitServer) getCachedReleases() ([]githubRelease, error) {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	if !s.cache.fetchedAt.IsZero() && time.Since(s.cache.fetchedAt) < releasesCacheTTL {
		s.log.Debug().Msg("releases served from cache")
		return s.cache.releases, nil
	}

	releases, err := s.fetchReleases()
	if err != nil {
		return nil, err
	}

	s.cache.releases = releases
	s.cache.fetchedAt = time.Now()
	s.log.Debug().Int("count", len(releases)).Msg("releases cache refreshed")
	return releases, nil
}

func (s *CockpitServer) fetchReleases() ([]githubRelease, error) {
	url := fmt.Sprintf(githubReleasesURL, s.opts.GitHub.Owner, s.opts.GitHub.Repo, releasesPerPage)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", "mir-cockpit")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decoding GitHub response: %w", err)
	}
	return releases, nil
}

func formatReleasesMarkdown(releases []githubRelease) string {
	if len(releases) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, r := range releases {
		fmt.Fprintf(&sb, "# %s — %s\n\n", r.TagName, r.PublishedAt.Format("2006-01-02"))
		sb.WriteString(r.Body)
		if i < len(releases)-1 {
			sb.WriteString("\n\n---\n\n")
		}
	}
	return sb.String()
}
