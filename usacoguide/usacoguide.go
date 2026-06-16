// Package usacoguide is the library behind the usacoguide command line:
// the HTTP client, request shaping, and typed data models for the USACO Guide.
//
// The guide's source lives at github.com/cpinitiative/usaco-guide. This client
// uses the GitHub API to list directories and fetches raw MDX files to parse
// YAML frontmatter. No GitHub token is required (60 req/hr unauthenticated).
package usacoguide

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultUserAgent identifies the client to the GitHub API.
const DefaultUserAgent = "usacoguide-cli/dev (+https://github.com/tamnd/usacoguide-cli)"

// Host is the usaco.guide site host.
const Host = "usaco.guide"

// GuideURL is the USACO Guide web URL.
const GuideURL = "https://usaco.guide"

// Config holds constructor parameters for the Client.
type Config struct {
	GitHubAPIURL string
	RawGitHubURL string
	UserAgent    string
	Rate         time.Duration
	Timeout      time.Duration
	Retries      int
}

// DefaultConfig returns sensible production defaults.
func DefaultConfig() Config {
	return Config{
		GitHubAPIURL: "https://api.github.com/repos/cpinitiative/usaco-guide/contents/content",
		RawGitHubURL: "https://raw.githubusercontent.com/cpinitiative/usaco-guide/main/content",
		UserAgent:    DefaultUserAgent,
		Rate:         300 * time.Millisecond,
		Timeout:      30 * time.Second,
		Retries:      3,
	}
}

// divisions maps the friendly division name to the directory name in the repo.
var divisions = map[string]string{
	"general":  "0_General",
	"bronze":   "2_Bronze",
	"silver":   "3_Silver",
	"gold":     "4_Gold",
	"platinum": "5_Plat",
	"advanced": "6_Advanced",
}

// divisionOrder is the canonical order of divisions.
var divisionOrder = []string{"general", "bronze", "silver", "gold", "platinum", "advanced"}

// guidePathFor maps division to the URL path used on usaco.guide.
var guidePathFor = map[string]string{
	"general":  "general",
	"bronze":   "bronze",
	"silver":   "silver",
	"gold":     "gold",
	"platinum": "plat",
	"advanced": "adv",
}

// Client talks to the GitHub API to read USACO Guide content.
type Client struct {
	cfg        Config
	httpClient *http.Client
	last       time.Time
}

// NewClient returns a Client ready to use.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// Modules lists all modules, optionally filtered by division.
func (c *Client) Modules(ctx context.Context, division string, limit int) ([]Module, error) {
	var divs []string
	if division == "" {
		divs = divisionOrder
	} else {
		div := strings.ToLower(division)
		if _, ok := divisions[div]; !ok {
			return nil, fmt.Errorf("unknown division %q; valid: %s", division, strings.Join(divisionOrder, ", "))
		}
		divs = []string{div}
	}

	var result []Module
	rank := 0
	for _, div := range divs {
		dirName := divisions[div]
		files, err := c.listDirectory(ctx, dirName)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if f.Type != "file" || !strings.HasSuffix(f.Name, ".mdx") {
				continue
			}
			id := strings.TrimSuffix(f.Name, ".mdx")
			// Fetch raw MDX to parse frontmatter.
			rawURL := c.cfg.RawGitHubURL + "/" + dirName + "/" + f.Name
			body, err := c.get(ctx, rawURL)
			if err != nil {
				// Skip if we can't fetch (rate limit, etc.).
				continue
			}
			fm := ParseFrontmatter(body)
			title := fm["title"]
			if title == "" {
				title = id
			}
			author := fm["author"]
			guidePath := guidePathFor[div]
			url := GuideURL + "/" + guidePath + "/" + id

			rank++
			result = append(result, Module{
				Rank:     rank,
				ID:       id,
				Title:    title,
				Division: div,
				Author:   author,
				URL:      url,
			})
			if limit > 0 && len(result) >= limit {
				return result, nil
			}
		}
	}
	return result, nil
}

// GetModule fetches a single module by ID, searching all divisions.
func (c *Client) GetModule(ctx context.Context, id string) (*Module, error) {
	for _, div := range divisionOrder {
		dirName := divisions[div]
		files, err := c.listDirectory(ctx, dirName)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.Type != "file" {
				continue
			}
			fileID := strings.TrimSuffix(f.Name, ".mdx")
			if fileID != id {
				continue
			}
			rawURL := c.cfg.RawGitHubURL + "/" + dirName + "/" + f.Name
			body, err := c.get(ctx, rawURL)
			if err != nil {
				return nil, err
			}
			fm := ParseFrontmatter(body)
			title := fm["title"]
			if title == "" {
				title = id
			}
			guidePath := guidePathFor[div]
			return &Module{
				Rank:     1,
				ID:       id,
				Title:    title,
				Division: div,
				Author:   fm["author"],
				URL:      GuideURL + "/" + guidePath + "/" + id,
			}, nil
		}
	}
	return nil, fmt.Errorf("module %q not found", id)
}

// Info returns aggregate statistics about the guide.
func (c *Client) Info(ctx context.Context) (Info, error) {
	total := 0
	for _, div := range divisionOrder {
		dirName := divisions[div]
		files, err := c.listDirectory(ctx, dirName)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.Type == "file" && strings.HasSuffix(f.Name, ".mdx") {
				total++
			}
		}
	}
	return Info{
		TotalModules: total,
		Divisions:    divisionOrder,
		SourceURL:    "https://github.com/cpinitiative/usaco-guide",
		GuideURL:     GuideURL,
	}, nil
}

// listDirectory fetches the GitHub API directory listing for a subdirectory.
func (c *Client) listDirectory(ctx context.Context, subdir string) ([]githubFile, error) {
	url := c.cfg.GitHubAPIURL + "/" + subdir
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	var files []githubFile
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("parse directory listing for %s: %w", subdir, err)
	}
	return files, nil
}

// ParseFrontmatter parses YAML frontmatter between "---" delimiters.
// It returns a flat map of key: value pairs (string values only).
func ParseFrontmatter(content []byte) map[string]string {
	result := map[string]string{}
	s := string(content)
	if !strings.HasPrefix(s, "---") {
		return result
	}
	// Find the closing ---.
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return result
	}
	fm := rest[:end]
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes.
		val = strings.Trim(val, `"'`)
		if key != "" {
			result[key] = val
		}
	}
	return result
}

// get fetches a URL and returns the body bytes with pacing and retry.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace enforces the inter-request rate limit.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		return 5 * time.Second
	}
	return d
}

// nodeText is kept for interface compatibility but unused in this package.
var _ = bytes.NewReader
