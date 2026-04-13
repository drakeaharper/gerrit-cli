package utils

import "testing"

func TestValidateChangeID(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"12345", false},
		{"1", false},
		{"I0123456789abcdef0123456789abcdef01234567", false},
		{"", true},
		{"abc", true},
		{"I0123", true}, // too short for full change ID
		{"I0123456789abcdef0123456789abcdef0123456g", true}, // non-hex char
	}

	for _, tt := range tests {
		err := ValidateChangeID(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateChangeID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"main", false},
		{"feature/test", false},
		{"release-1.0", false},
		{"some_branch.name", false},
		{"", true},
		{"../escape", true},
		{"/leading", true},
		{"trailing/", true},
		{"has space", true},
	}

	for _, tt := range tests {
		err := ValidateBranchName(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateBranchName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestValidateServerURL(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"gerrit.example.com", false},
		{"https://gerrit.example.com", false},
		{"http://gerrit.example.com", false},
		{"ssh://gerrit.example.com", false},
		{"", true},
		{"ftp://gerrit.example.com", true},
		{"gerrit.example.com; rm -rf /", true},
	}

	for _, tt := range tests {
		err := ValidateServerURL(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateServerURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		input   int
		wantErr bool
	}{
		{80, false},
		{443, false},
		{29418, false},
		{65535, false},
		{1, false},
		{0, true},
		{-1, true},
		{65536, true},
	}

	for _, tt := range tests {
		err := ValidatePort(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidatePort(%d) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"john.doe", false},
		{"admin", false},
		{"user@example.com", false}, // @ and . are allowed
		{"", true},
		{"user;rm", true},
		{"user|pipe", true},
	}

	for _, tt := range tests {
		err := ValidateUsername(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateUsername(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"file.txt", "file.txt", false},
		{"my-file_2.json", "my-file_2.json", false},
		{"", "", true},
		{"../../etc/passwd", "passwd", false}, // Base strips directory
	}

	for _, tt := range tests {
		got, err := SanitizeFilename(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("SanitizeFilename(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
