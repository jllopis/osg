package content

import "testing"

func TestBuildOutputPath(t *testing.T) {
	path := BuildOutputPath("content", "{date}/{slug}", "2025/11/02", "my-post")
	if path != "content/2025/11/02/my-post/index.md" {
		t.Fatalf("unexpected path: %s", path)
	}
}
