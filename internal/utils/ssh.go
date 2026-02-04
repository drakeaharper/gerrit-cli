package utils

import (
	"bufio"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// CreateSecureHostKeyCallback creates a host key callback that verifies known hosts
// and prompts for new hosts instead of blindly accepting them
func CreateSecureHostKeyCallback() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Get the known_hosts file path
		knownHostsPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")

		// Check if key is in known_hosts
		if isHostKeyKnown(knownHostsPath, hostname, key) {
			return nil
		}

		// Key is not known - show warning and require user confirmation
		keyType := key.Type()
		fingerprint := ssh.FingerprintSHA256(key)

		fmt.Fprintf(os.Stderr, "\nWarning: The authenticity of host '%s' can't be established.\n", hostname)
		fmt.Fprintf(os.Stderr, "%s key fingerprint is %s\n", keyType, fingerprint)
		fmt.Fprintf(os.Stderr, "Are you sure you want to continue connecting? This will add the key to known_hosts.\n")
		fmt.Fprintf(os.Stderr, "Type 'yes' to continue: ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}

		if strings.ToLower(response) != "yes" {
			return fmt.Errorf("host key verification failed")
		}

		// Add the key to known_hosts
		if err := addHostKey(knownHostsPath, hostname, key); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add host key to known_hosts: %v\n", err)
		}

		return nil
	}
}

// isHostKeyKnown checks if a host key is in the known_hosts file
func isHostKeyKnown(knownHostsPath, hostname string, key ssh.PublicKey) bool {
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		return false
	}

	file, err := os.Open(knownHostsPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read and parse known_hosts file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line: hostname keytype keydata
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		hostPart := parts[0]
		keyType := parts[1]
		keyData := parts[2]

		// Check if hostname matches (simple check)
		if !strings.Contains(hostPart, hostname) {
			continue
		}

		// Parse the stored key
		storedKeyBytes, err := base64.StdEncoding.DecodeString(keyData)
		if err != nil {
			continue
		}

		storedKey, err := ssh.ParsePublicKey(storedKeyBytes)
		if err != nil {
			continue
		}

		// Compare keys
		if keyType == key.Type() && string(storedKey.Marshal()) == string(key.Marshal()) {
			return true
		}
	}

	return scanner.Err() == nil && false
}

// addHostKey adds a host key to the known_hosts file
func addHostKey(knownHostsPath, hostname string, key ssh.PublicKey) error {
	// Ensure .ssh directory exists
	sshDir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Open known_hosts file for appending
	file, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

	// Format the host key entry
	keyData := base64.StdEncoding.EncodeToString(key.Marshal())
	entry := fmt.Sprintf("%s %s %s\n", hostname, key.Type(), keyData)

	// Write the entry
	if _, err := file.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write host key: %w", err)
	}

	return nil
}

// GetSSHKeyFingerprint returns a human-readable fingerprint of an SSH key
func GetSSHKeyFingerprint(key ssh.PublicKey) string {
	return ssh.FingerprintSHA256(key)
}

// ValidateSSHKey performs basic validation on an SSH key
// Note: This only checks file existence and readability, not key parsing.
// Passphrase-protected keys are handled by the system ssh command via ssh-agent.
func ValidateSSHKey(keyPath string) error {
	if keyPath == "" {
		return fmt.Errorf("SSH key path cannot be empty")
	}

	// Check if file exists
	info, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("SSH key file does not exist: %s", keyPath)
	}
	if err != nil {
		return fmt.Errorf("failed to access SSH key file: %w", err)
	}

	// Check it's a file, not a directory
	if info.IsDir() {
		return fmt.Errorf("SSH key path is a directory, not a file: %s", keyPath)
	}

	return nil
}

// GetSSHKeyType returns the type of SSH key (rsa, ed25519, etc.)
func GetSSHKeyType(keyPath string) (string, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SSH key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to parse SSH key: %w", err)
	}

	publicKey := signer.PublicKey()

	switch publicKey.Type() {
	case ssh.KeyAlgoRSA:
		if rsaKey, ok := publicKey.(ssh.CryptoPublicKey); ok {
			if cryptoKey, ok := rsaKey.CryptoPublicKey().(*rsa.PublicKey); ok {
				bitSize := cryptoKey.Size() * 8
				return fmt.Sprintf("RSA %d", bitSize), nil
			}
		}
		return "RSA", nil
	case ssh.KeyAlgoED25519:
		return "ED25519", nil
	case ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521:
		return "ECDSA", nil
	default:
		return publicKey.Type(), nil
	}
}
