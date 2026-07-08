package cmd

import "testing"

// A reply/resolve payload must point in_reply_to at the thread's LAST comment
// (so it threads) and carry the thread ROOT's file+line as coordinates. The
// revision it is posted against (the current revision) is chosen by
// postThreadReply, deliberately NOT the parent comment's patch set — see the
// "file ... not found in revision" regression (change 416074).
func TestBuildThreadReplyPayload(t *testing.T) {
	thread := []Comment{
		{ID: "root", PatchSet: 4, File: "a/b.tsx", Line: 40},   // reviewer's comment, anchored to PS4
		{ID: "reply1", PatchSet: 4, File: "a/b.tsx", Line: 40}, // an earlier reply
	}

	file, reply, err := buildThreadReply(thread, "explanation", boolPtr(false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file != "a/b.tsx" {
		t.Fatalf("expected file a/b.tsx (root), got %q", file)
	}
	if reply.InReplyTo != "reply1" {
		t.Errorf("expected in_reply_to=reply1 (last comment), got %q", reply.InReplyTo)
	}
	if reply.Line != 40 {
		t.Errorf("expected line 40 (root), got %d", reply.Line)
	}
	if reply.Message != "explanation" {
		t.Errorf("expected message to be carried through, got %q", reply.Message)
	}
	if reply.Unresolved == nil || *reply.Unresolved != false {
		t.Errorf("expected unresolved=false for resolve, got %v", reply.Unresolved)
	}
}

func TestBuildThreadReplyRequiresCommentID(t *testing.T) {
	if _, _, err := buildThreadReply([]Comment{{File: "a", Line: 1}}, "x", nil); err == nil {
		t.Fatal("expected error when comment ID is missing")
	}
}
