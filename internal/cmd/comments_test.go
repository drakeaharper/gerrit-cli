package cmd

import "testing"

// Two independent root threads anchored at the same (file, line), one of which
// is resolved via a reply. They must surface as two distinct threads with
// independent resolution state — not collapse into one.
func TestBuildCommentThreadsSharedLine(t *testing.T) {
	comments := []Comment{
		{ID: "root-suggestion", File: "f.rb", Line: 1202, Updated: "2026-06-29 10:00:00", Unresolved: true, Message: "use .includes(:enrollment_state)"},
		{ID: "root-n1", File: "f.rb", Line: 1202, Updated: "2026-06-29 10:05:00", Unresolved: true, Message: "warm-path N+1 here"},
		{ID: "reply-n1", File: "f.rb", Line: 1202, Updated: "2026-06-29 11:00:00", Unresolved: false, InReplyTo: "root-n1", Message: "fixed, thanks"},
	}

	threads := markThreadResolution(buildCommentThreads(comments))

	if len(threads) != 2 {
		t.Fatalf("expected 2 threads at shared line, got %d", len(threads))
	}

	byRoot := make(map[string][]Comment)
	for _, thread := range threads {
		if len(thread) == 0 {
			t.Fatal("empty thread")
		}
		byRoot[thread[0].ID] = thread
	}

	suggestion, ok := byRoot["root-suggestion"]
	if !ok {
		t.Fatal("suggestion thread missing")
	}
	if len(suggestion) != 1 {
		t.Fatalf("suggestion thread should have 1 comment, got %d", len(suggestion))
	}
	if !suggestion[0].Unresolved {
		t.Error("suggestion thread should be unresolved")
	}

	n1, ok := byRoot["root-n1"]
	if !ok {
		t.Fatal("N+1 thread missing")
	}
	if len(n1) != 2 {
		t.Fatalf("N+1 thread should have 2 comments (root + reply), got %d", len(n1))
	}
	if n1[0].Unresolved {
		t.Error("N+1 thread should be resolved (leaf reply resolved it)")
	}
}

// A reply whose parent is not present in the set (e.g. partial data) should
// still surface as its own root thread rather than vanishing.
func TestBuildCommentThreadsOrphanReply(t *testing.T) {
	comments := []Comment{
		{ID: "reply", File: "f.rb", Line: 10, Updated: "2026-06-29 10:00:00", Unresolved: true, InReplyTo: "missing-parent", Message: "orphan"},
	}

	threads := buildCommentThreads(comments)
	if len(threads) != 1 {
		t.Fatalf("expected 1 thread for orphan reply, got %d", len(threads))
	}
}

// Comments without IDs (SSH-sourced) fall back to (file, line) grouping.
func TestBuildCommentThreadsSSHFallback(t *testing.T) {
	comments := []Comment{
		{File: "f.rb", Line: 5, Updated: "2026-06-29 10:00:00", Message: "a"},
		{File: "f.rb", Line: 5, Updated: "2026-06-29 10:01:00", Message: "b"},
		{File: "f.rb", Line: 9, Updated: "2026-06-29 10:02:00", Message: "c"},
	}

	threads := buildCommentThreads(comments)
	if len(threads) != 2 {
		t.Fatalf("expected 2 line-grouped threads, got %d", len(threads))
	}
}
