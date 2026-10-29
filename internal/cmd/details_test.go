package cmd

import "testing"

func vote(value float64, name string) map[string]interface{} {
	return map[string]interface{}{"value": value, "name": name}
}

func account(id float64, name string) map[string]interface{} {
	return map[string]interface{}{"_account_id": id, "name": name}
}

// A label that already carries a verdict still has to show every score: the
// "approved" / "rejected" entries are AccountInfo objects without a "value",
// so consulting them first used to print an empty line.
func TestFormatLabelVotesApprovedLabel(t *testing.T) {
	data := map[string]interface{}{
		"approved": account(1001, "Reviewer"),
		"all":      []interface{}{vote(2, "Reviewer"), vote(-1, "Voter")},
	}

	got := formatLabelVotes("Code-Review", data)
	want := "+2 by Reviewer, -1 by Voter"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatLabelVotesRejectedLabel(t *testing.T) {
	data := map[string]interface{}{
		"rejected": account(1002, "Voter"),
		"all":      []interface{}{vote(-2, "Voter")},
	}

	got := formatLabelVotes("Code-Review", data)
	want := "-2 by Voter"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatLabelVotesNoVerdict(t *testing.T) {
	data := map[string]interface{}{
		"all": []interface{}{vote(0, "Voter")},
	}

	got := formatLabelVotes("Verified", data)
	want := "0 by Voter"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// Responses without "all" still render a readable line.
func TestFormatLabelVotesFallsBackToVerdict(t *testing.T) {
	approved := map[string]interface{}{"approved": account(1001, "Reviewer")}
	if got, want := formatLabelVotes("Verified", approved), "approved by Reviewer"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	rejected := map[string]interface{}{"rejected": account(1002, "Voter")}
	if got, want := formatLabelVotes("Code-Review", rejected), "rejected by Voter"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatLabelVotesEmpty(t *testing.T) {
	if got := formatLabelVotes("Verified", map[string]interface{}{}); got != "no votes" {
		t.Fatalf("got %q, want %q", got, "no votes")
	}
}
