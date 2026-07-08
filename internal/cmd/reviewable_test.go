package cmd

import (
	"testing"

	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
)

// restLabel builds a DETAILED_LABELS-style label map with the given vote values.
func restLabel(values ...int) map[string]interface{} {
	all := make([]interface{}, 0, len(values))
	for _, v := range values {
		all = append(all, map[string]interface{}{"value": float64(v)})
	}
	return map[string]interface{}{"all": all}
}

func changeWithLabels(labels map[string]interface{}) gerrit.Change {
	return gerrit.Change{Labels: labels}
}

func TestIsReviewable(t *testing.T) {
	cases := []struct {
		name   string
		change gerrit.Change
		want   bool
	}{
		{"no votes at all", changeWithLabels(nil), true},
		{"positive CR", changeWithLabels(map[string]interface{}{"Code-Review": restLabel(2)}), true},
		{"CR -1 blocks", changeWithLabels(map[string]interface{}{"Code-Review": restLabel(-1)}), false},
		{"CR -2 blocks", changeWithLabels(map[string]interface{}{"Code-Review": restLabel(-2)}), false},
		{"mixed CR +2 and -1 blocks", changeWithLabels(map[string]interface{}{"Code-Review": restLabel(2, -1)}), false},
		{"QA -1 blocks", changeWithLabels(map[string]interface{}{"QA-Review": restLabel(-1)}), false},
		{"QA +1", changeWithLabels(map[string]interface{}{"QA-Review": restLabel(1)}), true},
		{"Lint -2 blocks", changeWithLabels(map[string]interface{}{"Lint-Review": restLabel(-2)}), false},
		{"Lint -1 does not block", changeWithLabels(map[string]interface{}{"Lint-Review": restLabel(-1)}), true},
		{"Lint +1", changeWithLabels(map[string]interface{}{"Lint-Review": restLabel(1)}), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isReviewable(tc.change); got != tc.want {
				t.Fatalf("isReviewable = %v, want %v", got, tc.want)
			}
		})
	}
}

// SSH-format changes carry votes in currentPatchSet.approvals instead of labels.
func TestIsReviewableSSHFormat(t *testing.T) {
	blocked := gerrit.Change{CurrentPatchSet: &gerrit.SSHPatchSet{
		Approvals: []gerrit.ApprovalInfo{{Type: "Code-Review", Value: -1}},
	}}
	if isReviewable(blocked) {
		t.Fatal("SSH change with Code-Review -1 should not be reviewable")
	}

	ok := gerrit.Change{CurrentPatchSet: &gerrit.SSHPatchSet{
		Approvals: []gerrit.ApprovalInfo{{Type: "Code-Review", Value: 2}},
	}}
	if !isReviewable(ok) {
		t.Fatal("SSH change with Code-Review +2 should be reviewable")
	}
}

func TestPartitionReviewable(t *testing.T) {
	changes := []gerrit.Change{
		changeWithLabels(map[string]interface{}{"Code-Review": restLabel(2)}),  // reviewable
		changeWithLabels(map[string]interface{}{"Code-Review": restLabel(-1)}), // not
		changeWithLabels(map[string]interface{}{"Lint-Review": restLabel(-2)}), // not
		changeWithLabels(nil), // reviewable
	}

	reviewable, notReviewable := partitionReviewable(changes)
	if len(reviewable) != 2 {
		t.Fatalf("expected 2 reviewable, got %d", len(reviewable))
	}
	if len(notReviewable) != 2 {
		t.Fatalf("expected 2 not reviewable, got %d", len(notReviewable))
	}
}
