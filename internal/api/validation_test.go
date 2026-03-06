package api

import (
	"strings"
	"testing"
)

func TestPageViewRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     PageViewRequest
		wantErr string
	}{
		{"valid", PageViewRequest{Path: "/posts/hello", Fingerprint: "abc"}, ""},
		{"empty path", PageViewRequest{Path: "", Fingerprint: "abc"}, "path is required"},
		{"no leading slash", PageViewRequest{Path: "posts/hello", Fingerprint: "abc"}, "path must start with /"},
		{"empty fingerprint", PageViewRequest{Path: "/posts/hello", Fingerprint: ""}, "fp (fingerprint) is required"},
		{"long fingerprint", PageViewRequest{Path: "/posts/hello", Fingerprint: strings.Repeat("x", 129)}, "fp too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestVoteRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     VoteRequest
		wantErr string
	}{
		{"like", VoteRequest{Path: "/a", Fingerprint: "fp", Vote: 1}, ""},
		{"dislike", VoteRequest{Path: "/a", Fingerprint: "fp", Vote: -1}, ""},
		{"retract", VoteRequest{Path: "/a", Fingerprint: "fp", Vote: 0}, ""},
		{"invalid vote 2", VoteRequest{Path: "/a", Fingerprint: "fp", Vote: 2}, "vote must be"},
		{"invalid vote -2", VoteRequest{Path: "/a", Fingerprint: "fp", Vote: -2}, "vote must be"},
		{"empty path", VoteRequest{Path: "", Fingerprint: "fp", Vote: 1}, "path is required"},
		{"empty fp", VoteRequest{Path: "/a", Fingerprint: "", Vote: 1}, "fp (fingerprint) is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}
