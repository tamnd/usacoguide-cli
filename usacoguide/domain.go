package usacoguide

import (
	"context"
	"fmt"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes usacoguide as a kit Domain so a multi-domain host (ant)
// can enable it with a single blank import:
//
//	import _ "github.com/tamnd/usacoguide-cli/usacoguide"
//
// The same Domain builds the standalone usacoguide binary (see cli/root.go).
func init() { kit.Register(Domain{}) }

// Domain is the usacoguide driver. It carries no state.
type Domain struct{}

// Info describes the scheme, accepted hostnames, and the binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "usacoguide",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "usacoguide",
			Short:  "Browse USACO Guide modules from the command line",
			Long: `usacoguide reads module data from the USACO Guide (usaco.guide) via the
GitHub API. It lists modules by division, fetches YAML frontmatter from MDX
source files, and constructs usaco.guide links. No API key required
(GitHub unauthenticated limit: 60 requests/hour).

Commands:
  list       List modules (optionally filtered by --division)
  module     Show a specific module by ID
  export     Export all modules for a division (or all divisions)
  info       Show aggregate statistics

Divisions: general, bronze, silver, gold, platinum, advanced

usacoguide is an independent tool and is not affiliated with the USACO Guide.`,
			Site: Host,
			Repo: "https://github.com/tamnd/usacoguide-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "list", Group: "read", List: true,
		Summary: "List modules (optionally filtered by division)"}, listModules)

	kit.Handle(app, kit.OpMeta{Name: "module", Group: "read", Single: true,
		Summary: "Show a specific module by ID",
		Args:    []kit.Arg{{Name: "id", Help: "module ID, e.g. time-complexity"}}}, getModule)

	kit.Handle(app, kit.OpMeta{Name: "export", Group: "read", List: true,
		Summary: "Export all modules for a division (or all divisions)"}, exportModules)

	kit.Handle(app, kit.OpMeta{Name: "info", Group: "read", Single: true,
		Summary: "Show aggregate statistics about the guide"}, getInfo)
}

// newClient builds the Client from the kit config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- list ---

type listModulesIn struct {
	Division string  `kit:"flag"        help:"filter by division (general, bronze, silver, gold, platinum, advanced)"`
	Limit    int     `kit:"flag,inherit" help:"max results (0 = no limit)"`
	Client   *Client `kit:"inject"`
}

func listModules(ctx context.Context, in listModulesIn, emit func(*Module) error) error {
	items, err := in.Client.Modules(ctx, in.Division, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

// --- module ---

type getModuleIn struct {
	ID     string  `kit:"arg"    help:"module ID, e.g. time-complexity"`
	Client *Client `kit:"inject"`
}

func getModule(ctx context.Context, in getModuleIn, emit func(*Module) error) error {
	if in.ID == "" {
		return errs.Usage("id is required")
	}
	m, err := in.Client.GetModule(ctx, in.ID)
	if err != nil {
		return mapErr(err)
	}
	return emit(m)
}

// --- export ---

type exportModulesIn struct {
	Division string  `kit:"flag"   help:"export modules for a specific division (empty = all)"`
	Client   *Client `kit:"inject"`
}

func exportModules(ctx context.Context, in exportModulesIn, emit func(*Module) error) error {
	items, err := in.Client.Modules(ctx, in.Division, 0)
	if err != nil {
		return mapErr(err)
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

// --- info ---

type getInfoIn struct {
	Client *Client `kit:"inject"`
}

func getInfo(ctx context.Context, in getInfoIn, emit func(*Info) error) error {
	info, err := in.Client.Info(ctx)
	if err != nil {
		return mapErr(err)
	}
	return emit(&info)
}

// Classify turns a module ID or URL into (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "https://usaco.guide/") {
		// Extract the module ID from the URL path.
		path := strings.TrimPrefix(input, "https://usaco.guide/")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return "module", parts[len(parts)-1], nil
		}
	}
	if input != "" {
		return "module", input, nil
	}
	return "", "", errs.Usage("cannot classify %q: use a module ID or usaco.guide URL", input)
}

// Locate is the inverse of Classify.
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "module":
		return fmt.Sprintf("%s/%s", GuideURL, id), nil
	default:
		return "", errs.Usage("usacoguide has no resource type %q", uriType)
	}
}

// mapErr converts errors to kit kinds.
func mapErr(err error) error {
	return err
}
