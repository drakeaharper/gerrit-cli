package utils

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// Regular expressions for validation
var (
	// Gerrit change ID format: I followed by 40 hex characters
	changeIDRegex = regexp.MustCompile(`^I[0-9a-fA-F]{40}$`)
	
	// Git branch name validation (simplified but secure)
	// Allows alphanumeric, dash, underscore, slash, and dot
	branchNameRegex = regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)
	
	// Gerrit change number format
	changeNumberRegex = regexp.MustCompile(`^\d+$`)
	
	// Safe filename characters
	safeFilenameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// ValidateChangeID validates a Gerrit change ID
func ValidateChangeID(changeID string) error {
	if changeID == "" {
		return fmt.Errorf("change ID cannot be empty")
	}
	
	// Check if it's a change number
	if changeNumberRegex.MatchString(changeID) {
		return nil
	}
	
	// Check if it's a full change ID
	if !changeIDRegex.MatchString(changeID) {
		return fmt.Errorf("invalid change ID format: %s", changeID)
	}
	
	return nil
}

// ValidateBranchName validates a git branch name
func ValidateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	
	if len(branch) > 255 {
		return fmt.Errorf("branch name too long")
	}
	
	// Check for dangerous patterns
	if strings.Contains(branch, "..") {
		return fmt.Errorf("branch name cannot contain '..'")
	}
	
	if strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
		return fmt.Errorf("branch name cannot start or end with '/'")
	}
	
	if !branchNameRegex.MatchString(branch) {
		return fmt.Errorf("branch name contains invalid characters: %s", branch)
	}
	
	return nil
}

// ValidateServerURL validates a server URL
func ValidateServerURL(serverURL string) error {
	if serverURL == "" {
		return fmt.Errorf("server URL cannot be empty")
	}
	
	// Parse as URL if it contains protocol
	if strings.Contains(serverURL, "://") {
		u, err := url.Parse(serverURL)
		if err != nil {
			return fmt.Errorf("invalid server URL: %w", err)
		}
		
		if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ssh" {
			return fmt.Errorf("unsupported protocol: %s", u.Scheme)
		}
		
		if u.Host == "" {
			return fmt.Errorf("server URL missing host")
		}
	} else {
		// Just a hostname, validate it's not empty and doesn't contain suspicious chars
		if strings.ContainsAny(serverURL, " \t\n\r;|&$`") {
			return fmt.Errorf("server name contains invalid characters")
		}
	}
	
	return nil
}

// ValidatePort validates a port number
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

// ValidateAndCleanPath validates and cleans a file path to prevent traversal attacks
func ValidateAndCleanPath(basePath, userPath string) (string, error) {
	if userPath == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	
	// Clean the path
	cleanPath := filepath.Clean(userPath)
	
	// If it's an absolute path, ensure it's under the base path
	if filepath.IsAbs(cleanPath) {
		rel, err := filepath.Rel(basePath, cleanPath)
		if err != nil {
			return "", fmt.Errorf("invalid path: %w", err)
		}
		
		// Check if path tries to escape base directory
		if strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("path traversal attempt detected")
		}
		
		return cleanPath, nil
	}
	
	// For relative paths, join with base and clean
	fullPath := filepath.Join(basePath, cleanPath)
	cleanFullPath := filepath.Clean(fullPath)
	
	// Verify the cleaned path is still under base path
	rel, err := filepath.Rel(basePath, cleanFullPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal attempt detected")
	}
	
	return cleanFullPath, nil
}

// SanitizeFilename ensures a filename is safe for filesystem operations
func SanitizeFilename(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}
	
	// Remove any directory separators
	filename = filepath.Base(filename)
	
	// Check against safe pattern
	if !safeFilenameRegex.MatchString(filename) {
		// Try to clean it
		cleaned := regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(filename, "_")
		if cleaned == "" || cleaned == "." || cleaned == ".." {
			return "", fmt.Errorf("filename contains only invalid characters")
		}
		return cleaned, nil
	}
	
	return filename, nil
}

// ValidateUsername validates a username
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	
	if len(username) > 255 {
		return fmt.Errorf("username too long")
	}
	
	// Check for shell metacharacters
	if strings.ContainsAny(username, " \t\n\r;|&$`<>(){}[]\\\"'") {
		return fmt.Errorf("username contains invalid characters")
	}
	
	return nil
}