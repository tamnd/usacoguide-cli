package usacoguide

// Module is one module from the USACO Guide.
// Modules are MDX files in the cpinitiative/usaco-guide GitHub repository.
type Module struct {
	Rank     int    `json:"rank"`
	ID       string `json:"id"`       // filename without .mdx, e.g. "time-complexity"
	Title    string `json:"title"`
	Division string `json:"division"` // general, bronze, silver, gold, platinum, advanced
	Author   string `json:"author,omitempty"`
	Order    int    `json:"order,omitempty"`
	URL      string `json:"url"` // usaco.guide URL
}

// githubFile is one entry from the GitHub API directory listing.
type githubFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "file" or "dir"
}

// Info holds aggregate information about the guide.
type Info struct {
	TotalModules int      `json:"total_modules"`
	Divisions    []string `json:"divisions"`
	SourceURL    string   `json:"source_url"`
	GuideURL     string   `json:"guide_url"`
}
