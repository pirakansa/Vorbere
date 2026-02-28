package manifest

import (
	"reflect"
	"testing"
)

func TestFindExactArchiveEntry(t *testing.T) {
	entries := []archiveEntry{
		{path: "pkg/bin/tool", body: []byte("tool"), mode: 0o755},
		{path: "pkg/README.md", body: []byte("readme"), mode: 0o644},
	}

	got := findExactArchiveEntry(entries, "pkg/bin/tool")
	if got == nil {
		t.Fatalf("expected exact entry match")
	}
	if string(got.content) != "tool" {
		t.Fatalf("unexpected content: %q", string(got.content))
	}
	if got.mode != 0o755 {
		t.Fatalf("unexpected mode: %v", got.mode)
	}

	if miss := findExactArchiveEntry(entries, "pkg/bin/missing"); miss != nil {
		t.Fatalf("expected nil for missing extract path")
	}
}

func TestCollectArchiveChildren(t *testing.T) {
	entries := []archiveEntry{
		{path: "pkg/bin/tool", body: []byte("tool"), mode: 0o755},
		{path: "pkg/lib/help.md", body: []byte("help"), mode: 0o644},
		{path: "other/file.txt", body: []byte("other"), mode: 0o644},
	}

	got := collectArchiveChildren(entries, "pkg")
	wantPaths := []string{"bin/tool", "lib/help.md"}

	if len(got) != len(wantPaths) {
		t.Fatalf("expected %d children, got %d", len(wantPaths), len(got))
	}

	var gotPaths []string
	for _, entry := range got {
		gotPaths = append(gotPaths, entry.path)
	}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("unexpected child paths: got=%v want=%v", gotPaths, wantPaths)
	}
}

func TestArchiveApplySummaryOutcomePriority(t *testing.T) {
	summary := archiveApplySummary{}
	if got := summary.outcome(); got != outcomeUnchanged {
		t.Fatalf("expected unchanged on empty summary, got %q", got)
	}

	updateArchiveApplySummary(&summary, outcomeCreated)
	if got := summary.outcome(); got != outcomeCreated {
		t.Fatalf("expected created after created outcome, got %q", got)
	}

	updateArchiveApplySummary(&summary, outcomeUpdated)
	if got := summary.outcome(); got != outcomeUpdated {
		t.Fatalf("expected updated to take priority, got %q", got)
	}
}
