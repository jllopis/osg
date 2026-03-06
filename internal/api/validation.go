package api

import (
	"fmt"
	"strings"
)

// PageViewRequest is the JSON body for POST /api/v1/pageview.
type PageViewRequest struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fp"`
}

// VoteRequest is the JSON body for POST /api/v1/vote.
type VoteRequest struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fp"`
	Vote        int    `json:"vote"`
}

// Validate checks that a PageViewRequest has valid fields.
func (r PageViewRequest) Validate() error {
	if strings.TrimSpace(r.Path) == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(r.Path, "/") {
		return fmt.Errorf("path must start with /")
	}
	if strings.TrimSpace(r.Fingerprint) == "" {
		return fmt.Errorf("fp (fingerprint) is required")
	}
	if len(r.Fingerprint) > 128 {
		return fmt.Errorf("fp too long (max 128 chars)")
	}
	return nil
}

// Validate checks that a VoteRequest has valid fields.
func (r VoteRequest) Validate() error {
	if strings.TrimSpace(r.Path) == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(r.Path, "/") {
		return fmt.Errorf("path must start with /")
	}
	if strings.TrimSpace(r.Fingerprint) == "" {
		return fmt.Errorf("fp (fingerprint) is required")
	}
	if len(r.Fingerprint) > 128 {
		return fmt.Errorf("fp too long (max 128 chars)")
	}
	if r.Vote != -1 && r.Vote != 0 && r.Vote != 1 {
		return fmt.Errorf("vote must be -1, 0, or 1")
	}
	return nil
}
