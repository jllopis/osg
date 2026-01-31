package app

type CLIOptions struct {
	ConfigPath        string
	Verbose           bool
	DryRun            bool
	IncludeDrafts     *bool
	VaultPath         string
	ObsidianVaultBase string
	Vault             string
	OsgContentDir     string
}
