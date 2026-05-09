package ui

// ParamDef describes one form input on an operation card. The slice is
// looked up by operation name and rendered in the order given.
type ParamDef struct {
	Name        string   // form field name
	Label       string   // human label for the input
	Kind        string   // "bool" / "string" / "select"
	Required    bool     // marks the field with required
	Default     string   // default value for string/select inputs
	Placeholder string   // placeholder for string inputs
	Options     []string // choices for select inputs
	Help        string   // optional helper text below the input
}

// operationParamRegistry is the static schema for every operation that
// accepts user input. Operations not listed here get no form (the card
// only shows the Run button). Defaults match the CLI's behaviour when
// the corresponding flags are absent.
var operationParamRegistry = map[string][]ParamDef{
	"build": {
		{Name: "force-ai-summaries", Label: "Force AI summaries", Kind: "bool",
			Help: "Regenerate every AI summary, bypassing the cache (may incur API costs)."},
	},
	"deploy": {
		{Name: "provider", Label: "Provider", Kind: "select",
			Options: []string{"", "cloudflare", "rsync", "s3"},
			Help:    "Empty uses the deploy.provider from config.yaml."},
		{Name: "preview", Label: "Dry-run (validate only)", Kind: "bool"},
		{Name: "build", Label: "Build before deploying", Kind: "bool", Default: "true"},
	},
	"check": {
		{Name: "links", Label: "Check internal links", Kind: "bool"},
		{Name: "images", Label: "Check orphan images", Kind: "bool"},
		{Name: "frontmatter", Label: "Check frontmatter", Kind: "bool"},
		{Name: "json", Label: "JSON output", Kind: "bool"},
	},
	"audit": {
		{Name: "json", Label: "JSON output", Kind: "bool"},
	},
	"new": {
		{Name: "title", Label: "Title", Kind: "string", Required: true,
			Placeholder: "Why I switched to OSG"},
		{Name: "tags", Label: "Tags (comma-separated)", Kind: "string",
			Placeholder: "essay,tooling"},
		{Name: "publish", Label: "Mark as published", Kind: "bool",
			Help: "Default is draft."},
		{Name: "notes-dir", Label: "Notes subdirectory", Kind: "string",
			Help: "Optional override of new_notes_dir from config."},
	},
	"theme-init": {
		{Name: "name", Label: "Theme name", Kind: "string", Required: true,
			Placeholder: "my-theme"},
		{Name: "parent", Label: "Parent theme (inherits)", Kind: "string",
			Placeholder: "default"},
	},
	"plugin-install": {
		{Name: "path", Label: "Path or repo", Kind: "string", Required: true,
			Placeholder: "github.com/user/repo[@tag] or local .wasm path"},
	},
	"import-wordpress": {
		{Name: "file", Label: "WordPress WXR file", Kind: "string", Required: true,
			Placeholder: "/path/to/wordpress-export.xml"},
	},
	"import-hugo": {
		{Name: "dir", Label: "Hugo content directory", Kind: "string", Required: true,
			Placeholder: "/path/to/hugo-site/content"},
	},
}

// confirmOperations names the operations that should ask the user for
// confirmation before running. JS reads the data-confirm attribute on
// the form and shows a <dialog> modal.
var confirmOperations = map[string]string{
	"deploy":           "Deploy will push the built site to the configured remote. Continue?",
	"import-wordpress": "Import will create new files in the vault. Continue?",
	"import-hugo":      "Import will create new files in the vault. Continue?",
}

// operationPage assigns each operation to the page it should appear
// on. /actions only shows the small set of "global" tasks; the rest
// are surfaced in domain-specific pages (Vault for new, Plugins for
// install, Themes for theme-init, Import for the importers, Services
// for the long-running ones, Audit for audit). Operations not listed
// here default to "actions".
var operationPage = map[string]string{
	"init":             "actions",
	"build":            "actions",
	"deploy":           "actions",
	"update-content":   "actions",
	"check":            "actions",
	"audit":            "audit",
	"new":              "vault",
	"theme-init":       "themes",
	"plugin-install":   "plugins",
	"import-wordpress": "import",
	"import-hugo":      "import",
	"serve":            "services",
	"api":              "services",
	"watcher":          "services",
	"scheduler":        "services",
}

// actionFlow is the canonical pipeline rendered on /actions. Order
// matters: "Run from here" walks this list starting at the named
// operation and triggers each downstream step in sequence.
var actionFlow = []string{
	"init",
	"update-content",
	"check",
	"build",
	"deploy",
}

// flowDownstream returns the slice of operations from name onwards in
// actionFlow. Returns nil when name isn't part of the flow.
func flowDownstream(name string) []string {
	for i, n := range actionFlow {
		if n == name {
			return append([]string(nil), actionFlow[i:]...)
		}
	}
	return nil
}

// pageForOperation returns the dashboard page where an operation lives.
func pageForOperation(name string) string {
	if v, ok := operationPage[name]; ok {
		return v
	}
	return "actions"
}

// operationsForPage filters a slice of OperationView to those whose
// page is the given one. Stable order is preserved.
func operationsForPage(views []OperationView, page string) []OperationView {
	out := make([]OperationView, 0, len(views))
	for _, v := range views {
		if pageForOperation(v.Name) == page {
			out = append(out, v)
		}
	}
	return out
}

// collapsibleParams names operations whose param form should render
// inside a closed <details> element to keep the card compact.
var collapsibleParams = map[string]bool{
	"check": true,
}

// hasCollapsibleParams reports whether the named operation should
// have its parameters collapsed by default.
func hasCollapsibleParams(name string) bool {
	return collapsibleParams[name]
}

// paramsForOperation returns the schema for a given operation name (or
// nil when the operation has no parameters).
func paramsForOperation(name string) []ParamDef {
	return operationParamRegistry[name]
}

// confirmTextFor returns the modal text for an operation, or empty
// when no confirmation is required.
func confirmTextFor(name string) string {
	return confirmOperations[name]
}
