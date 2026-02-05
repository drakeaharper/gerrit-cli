package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test keys generated with ssh-keygen - DO NOT USE IN PRODUCTION
const (
	testKeyPassphraseProtected = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAACmFlczI1Ni1jdHIAAAAGYmNyeXB0AAAAGAAAABB5Pc0wDP
gVqGH/2e9BRJIbAAAAGAAAAAEAAAAzAAAAC3NzaC1lZDI1NTE5AAAAIGuQllm8jAYWYCBE
u3bEHWW150yTJw5YaJ6vAvDWJyoDAAAAoCQeQg9mCDxK9J1BKyjXnHtYp/tNlIJ1WCPneE
CltdjxPseva751D/M0b9ubbawxINxu86zLVYVlSoL5UYDsiXq8eeyfDPvUOa8tFsQZzY07
/mvJJ2F+a/ndE7sBSHfp0sMTzc2sBuiwS3qiEZHZlnJfAy7OWigaoGvo0BwD0MTzfbZshu
1N9i+6/jiW+grcdP+be4nduJMR+UOF4KMi0Aw=
-----END OPENSSH PRIVATE KEY-----
`
	testKeyRSA = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAABFwAAAAdzc2gtcn
NhAAAAAwEAAQAAAQEAxarnw0L/W8fKTkjo51uRkjmwBMJvVGnC7Ho0GuFQHguFW+U71Mvx
2ZrXgvwm4cbcCFYprVXg4uWDZmtDYd6H9FIJh7GbxAcnPPucWyM6QnQWQz65mGLqUgUfht
NQThl4cjk08VcJLUzt9tXKC3CNl4/5mO0UD0Hm4MCR6w4BJ7YBkJEIXikliQH3PFzCedXK
GKAZgQ/yhUPm73Pc036BD2NUqEdWmXySSXYOszj3H7jga44Ypn55uLOEl8Bnhy1CQk4J3V
pjDKH+XAlRHXTOKstWMMuLUnUCqIE7a3G+b30U/uYLswQST4aP5zI36D6VLMntxIlSYcQi
4c+apdQB/wAAA9DUn9901J/fdAAAAAdzc2gtcnNhAAABAQDFqufDQv9bx8pOSOjnW5GSOb
AEwm9UacLsejQa4VAeC4Vb5TvUy/HZmteC/CbhxtwIVimtVeDi5YNma0Nh3of0UgmHsZvE
Byc8+5xbIzpCdBZDPrmYYupSBR+G01BOGXhyOTTxVwktTO321coLcI2Xj/mY7RQPQebgwJ
HrDgEntgGQkQheKSWJAfc8XMJ51coYoBmBD/KFQ+bvc9zTfoEPY1SoR1aZfJJJdg6zOPcf
uOBrjhimfnm4s4SXwGeHLUJCTgndWmMMof5cCVEddM4qy1Ywy4tSdQKogTtrcb5vfRT+5g
uzBBJPho/nMjfoPpUsye3EiVJhxCLhz5ql1AH/AAAAAwEAAQAAAQB09lLbNHqbWVX5CqVd
uM4jYyUnO9HadhZUDV9lhGr+zDxmCvdjTCZYZ4ocRI3RTPUHrcxNd6JxP/OHl/KwJ5f01t
Iyy8JqtPzf1dZIC0k+5ygBNE1nwSf7znJAOiureuDNXdJY9/JDLuEkDI7YRApUY2oCtk4H
VSyDUw9Ese23C63G1iAx2dycWFxtMjT1IcjoeqRmRjI6BH/wms3Go7MokeChDaGGrN4LCc
LYIVuUnE7gt7hBBEReTJ9/bZcU4uZoCPBdxxa2LF6ql66UzF7+gLQ6ibkxWGPIOEuyQRrl
/jUmC6AkkOAWT8RpSRh/G57KwPqq1moP0oGNo6FMDruxAAAAgCIRV7ofTMB74I+avMRo1g
Zb/Fy9L56IOMFwI1F8JsbalqyLnV/TMTORR+94PkKvlUwjyz2n8/NI5xfjEQsnc+k0ReiX
CddnhAzOhd2YA2o3opztj+Rtr4lrz0iDVQL1Mk/8jM6bRmICg0gunlj2t69EpM7PaOo6gC
SncXFLe8NBAAAAgQDse2qbwCahCvszf/oiNPdqq3kSSshk6ljnp91QXoSpHx3Vqr45F1ZI
wZovd2cN5pPSi5d1NnGupS1iJdZKlWAacT/Y3OlQdaoxUl//DRgy2HB4jguGrQ+N/Mk9du
hlBaG1srSooXdiy4qJUdntdNLswip0Qm5RgBClgmoTgCe11wAAAIEA1ftikgjk/8ffAVS2
AIsQcDgIdscy0+PYvom1498IXkBJxBNddlYYVxB93FRoxUqZkVE/fi41rIOeIDM7ihIrMS
9BuFxpM0xo867oe5B/VQDUCcxaZXGOsc2oFTqAKmI5kXsLmvSnlRu4npiVN7U0Le+qKpM+
EkMJYtjIsjy+wBkAAAAYYWRyaWFuLmdydWJlckBHTlZQSjQyMlBNAQID
-----END OPENSSH PRIVATE KEY-----
`
)

func TestValidateSSHKey(t *testing.T) {
	t.Run("succeeds with a valid unencrypted key", func(t *testing.T) {
		keyPath := createTempKeyFile(t, testKeyRSA)

		err := ValidateSSHKey(keyPath)

		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("succeeds with a passphrase-protected key", func(t *testing.T) {
		keyPath := createTempKeyFile(t, testKeyPassphraseProtected)

		err := ValidateSSHKey(keyPath)

		if err != nil {
			t.Errorf("expected no error (key works via ssh-agent), got: %v", err)
		}
	})

	t.Run("returns an error when the path is empty", func(t *testing.T) {
		keyPath := ""

		err := ValidateSSHKey(keyPath)

		if err == nil {
			t.Error("expected an error for empty path")
		}
	})

	t.Run("returns an error when the file does not exist", func(t *testing.T) {
		keyPath := "/nonexistent/path/to/key"

		err := ValidateSSHKey(keyPath)

		if err == nil {
			t.Error("expected an error for non-existent file")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("expected 'does not exist' in error, got: %v", err)
		}
	})

	t.Run("returns an error when the SSH key is invalid", func(t *testing.T) {
		keyPath := createTempKeyFile(t, "not a valid ssh key")

		err := ValidateSSHKey(keyPath)

		if err == nil {
			t.Error("expected an error for invalid key format")
		}
		if !strings.Contains(err.Error(), "invalid SSH private key") {
			t.Errorf("expected 'invalid SSH private key' in error, got: %v", err)
		}
	})
}

func createTempKeyFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(keyPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test key file: %v", err)
	}
	return keyPath
}
