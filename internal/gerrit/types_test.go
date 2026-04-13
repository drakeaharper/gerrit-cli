package gerrit

import (
	"encoding/json"
	"testing"
)

func TestAccountDisplayName(t *testing.T) {
	tests := []struct {
		account Account
		want    string
	}{
		{Account{Name: "John Doe", Username: "john", Email: "john@example.com"}, "John Doe"},
		{Account{Username: "john", Email: "john@example.com"}, "john"},
		{Account{Email: "john@example.com"}, "john@example.com"},
		{Account{}, "unknown"},
	}

	for _, tt := range tests {
		got := tt.account.DisplayName()
		if got != tt.want {
			t.Errorf("Account%+v.DisplayName() = %q, want %q", tt.account, got, tt.want)
		}
	}
}

func TestChangeNumber(t *testing.T) {
	tests := []struct {
		change Change
		want   int
	}{
		{Change{Number: 12345}, 12345},
		{Change{NumberSSH: 67890}, 67890},
		{Change{Number: 12345, NumberSSH: 67890}, 12345}, // REST takes precedence
		{Change{}, 0},
	}

	for _, tt := range tests {
		got := tt.change.ChangeNumber()
		if got != tt.want {
			t.Errorf("Change{Number:%d, NumberSSH:%d}.ChangeNumber() = %d, want %d",
				tt.change.Number, tt.change.NumberSSH, got, tt.want)
		}
	}
}

func TestChangeUpdatedTime(t *testing.T) {
	c1 := Change{Updated: "2025-01-01 12:00:00"}
	if got := c1.UpdatedTime(); got != "2025-01-01 12:00:00" {
		t.Errorf("UpdatedTime() with Updated = %q, want %q", got, "2025-01-01 12:00:00")
	}

	c2 := Change{LastUpdated: 1704067200} // 2024-01-01 00:00:00 UTC
	got := c2.UpdatedTime()
	if got == "" {
		t.Error("UpdatedTime() with LastUpdated returned empty")
	}

	c3 := Change{}
	if got := c3.UpdatedTime(); got != "" {
		t.Errorf("UpdatedTime() with no data = %q, want empty", got)
	}
}

func TestChangePatchSetNumber(t *testing.T) {
	c1 := Change{
		CurrentRevision: "abc123",
		Revisions: map[string]RevisionInfo{
			"abc123": {Number: 3},
		},
	}
	if got := c1.CurrentPatchSetNumber(); got != 3 {
		t.Errorf("CurrentPatchSetNumber() REST = %d, want 3", got)
	}

	c2 := Change{
		CurrentPatchSet: &SSHPatchSet{Number: 5},
	}
	if got := c2.CurrentPatchSetNumber(); got != 5 {
		t.Errorf("CurrentPatchSetNumber() SSH = %d, want 5", got)
	}

	c3 := Change{}
	if got := c3.CurrentPatchSetNumber(); got != 0 {
		t.Errorf("CurrentPatchSetNumber() empty = %d, want 0", got)
	}
}

func TestChangeUnmarshalREST(t *testing.T) {
	jsonData := `{
		"_number": 12345,
		"project": "my-project",
		"branch": "main",
		"subject": "Fix the thing",
		"status": "NEW",
		"owner": {"name": "John", "_account_id": 1},
		"current_revision": "abc123",
		"revisions": {
			"abc123": {"_number": 2}
		},
		"labels": {
			"Code-Review": {"approved": {"name": "Jane"}}
		}
	}`

	var change Change
	if err := json.Unmarshal([]byte(jsonData), &change); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if change.Number != 12345 {
		t.Errorf("Number = %d, want 12345", change.Number)
	}
	if change.Subject != "Fix the thing" {
		t.Errorf("Subject = %q, want %q", change.Subject, "Fix the thing")
	}
	if change.Owner.Name != "John" {
		t.Errorf("Owner.Name = %q, want %q", change.Owner.Name, "John")
	}
	if rev, ok := change.Revisions["abc123"]; !ok || rev.Number != 2 {
		t.Errorf("Revisions[abc123].Number = %d, want 2", rev.Number)
	}
	if change.Labels == nil {
		t.Error("Labels is nil")
	}
}

func TestChangeUnmarshalSSH(t *testing.T) {
	jsonData := `{
		"number": 67890,
		"project": "my-project",
		"branch": "main",
		"subject": "SSH change",
		"status": "NEW",
		"owner": {"username": "jdoe"},
		"lastUpdated": 1704067200,
		"currentPatchSet": {
			"number": 3,
			"revision": "def456",
			"approvals": [
				{"type": "Code-Review", "value": 2, "name": "Reviewer"}
			]
		}
	}`

	var change Change
	if err := json.Unmarshal([]byte(jsonData), &change); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if change.ChangeNumber() != 67890 {
		t.Errorf("ChangeNumber() = %d, want 67890", change.ChangeNumber())
	}
	if change.Owner.DisplayName() != "jdoe" {
		t.Errorf("Owner.DisplayName() = %q, want %q", change.Owner.DisplayName(), "jdoe")
	}
	if change.CurrentPatchSet == nil {
		t.Fatal("CurrentPatchSet is nil")
	}
	if change.CurrentPatchSetNumber() != 3 {
		t.Errorf("CurrentPatchSetNumber() = %d, want 3", change.CurrentPatchSetNumber())
	}
	if len(change.CurrentPatchSet.Approvals) != 1 {
		t.Fatalf("len(Approvals) = %d, want 1", len(change.CurrentPatchSet.Approvals))
	}
	if change.CurrentPatchSet.Approvals[0].Value != 2 {
		t.Errorf("Approval value = %d, want 2", change.CurrentPatchSet.Approvals[0].Value)
	}
}
