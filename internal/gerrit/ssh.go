package gerrit

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	config *config.Config
}

func NewSSHClient(cfg *config.Config) *SSHClient {
	return &SSHClient{
		config: cfg,
	}
}

// ExecuteCommand executes a Gerrit command with proper argument handling
// Deprecated: Use ExecuteCommandArgs for better security
func (c *SSHClient) ExecuteCommand(command string) (string, error) {
	// Parse the command to extract individual arguments
	// This is kept for backward compatibility but should be avoided
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}
	return c.ExecuteCommandArgs(parts...)
}

// ExecuteCommandArgs executes a Gerrit command with properly separated arguments
func (c *SSHClient) ExecuteCommandArgs(args ...string) (string, error) {
	sshKeyPath := c.config.SSHKey
	if sshKeyPath == "" {
		sshKeyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	}

	// Build SSH command with proper argument separation
	sshArgs := []string{
		"-p", fmt.Sprintf("%d", c.config.Port),
		"-i", sshKeyPath,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=~/.ssh/known_hosts",
		fmt.Sprintf("%s@%s", c.config.User, c.config.Server),
		"gerrit",
	}
	// Append Gerrit command arguments
	sshArgs = append(sshArgs, args...)

	cmd := exec.Command("ssh", sshArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("SSH command failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func (c *SSHClient) TestConnection() error {
	output, err := c.ExecuteCommand("version")
	if err != nil {
		return fmt.Errorf("failed to connect to Gerrit: %w", err)
	}

	if !strings.Contains(output, "gerrit version") {
		return fmt.Errorf("unexpected response from Gerrit server")
	}

	return nil
}

// StreamCommand streams output from a Gerrit command
// Deprecated: Use StreamCommandArgs for better security
func (c *SSHClient) StreamCommand(command string, output io.Writer) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}
	return c.StreamCommandArgs(output, parts...)
}

// StreamCommandArgs streams output from a Gerrit command with properly separated arguments
func (c *SSHClient) StreamCommandArgs(output io.Writer, args ...string) error {
	sshKeyPath := c.config.SSHKey
	if sshKeyPath == "" {
		sshKeyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	}

	// Build SSH command with proper argument separation
	sshArgs := []string{
		"-p", fmt.Sprintf("%d", c.config.Port),
		"-i", sshKeyPath,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=~/.ssh/known_hosts",
		fmt.Sprintf("%s@%s", c.config.User, c.config.Server),
		"gerrit",
	}
	// Append Gerrit command arguments
	sshArgs = append(sshArgs, args...)

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdout = output
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// QueryChanges executes a Gerrit query and returns the output
func (c *SSHClient) QueryChanges(query string, options ...string) (string, error) {
	args := []string{"query", "--format=JSON"}
	args = append(args, options...)
	args = append(args, query)

	// Use the secure version with separate arguments
	return c.ExecuteCommandArgs(args...)
}

// GetChangeDetails fetches details for a specific change
func (c *SSHClient) GetChangeDetails(changeID string) (string, error) {
	return c.QueryChanges(changeID, "--current-patch-set", "--all-approvals", "--comments", "--files")
}

// GetVersion returns the Gerrit server version
func (c *SSHClient) GetVersion() (string, error) {
	return c.ExecuteCommandArgs("version")
}

// CreateSSHClientFromKey creates an SSH client using golang.org/x/crypto/ssh
// This is an alternative implementation that doesn't rely on the ssh command
func (c *SSHClient) CreateSSHClientFromKey() (*ssh.Client, error) {
	sshKeyPath := c.config.SSHKey
	if sshKeyPath == "" {
		sshKeyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	}

	key, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: c.config.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// Use secure host key verification
		HostKeyCallback: utils.CreateSecureHostKeyCallback(),
	}

	addr := fmt.Sprintf("%s:%d", c.config.Server, c.config.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	return client, nil
}