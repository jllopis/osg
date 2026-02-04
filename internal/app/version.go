package app

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func VersionInfo() string {
	return fmt.Sprintf("version: %s\ncommit:  %s\ndate:    %s", Version, Commit, Date)
}
