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
