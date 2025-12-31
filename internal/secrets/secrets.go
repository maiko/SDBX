// Package secrets handles secret generation and management for sdbx.
package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SecretFiles defines the secrets that sdbx manages
var SecretFiles = map[string]int{
	"authelia_jwt_secret.txt":             64,
	"authelia_session_secret.txt":         64,
	"authelia_storage_encryption_key.txt": 64,
	"authelia_oidc_hmac_secret.txt":       64,
	"vpn_password.txt":                    0, // User-provided
	"cloudflared_tunnel_token.txt":        0, // User-provided
	"sonarr_api_key.txt":                  32,
	"radarr_api_key.txt":                  32,
}

// GenerateRandomString generates a cryptographically secure random string
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateSecrets creates all required secret files
func GenerateSecrets(secretsDir string) error {
	// Create secrets directory
	if err := os.MkdirAll(secretsDir, 0700); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	for filename, length := range SecretFiles {
		path := filepath.Join(secretsDir, filename)

		// Skip if file exists
		if _, err := os.Stat(path); err == nil {
			continue
		}

		// Skip user-provided secrets (length 0)
		if length == 0 {
			// Create empty placeholder
			if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
				return fmt.Errorf("failed to create %s: %w", filename, err)
			}
			continue
		}

		// Generate random secret
		secret, err := GenerateRandomString(length)
		if err != nil {
			return err
		}

		if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}

// RotateSecret regenerates a specific secret and creates a backup
func RotateSecret(secretsDir, name string) (string, error) {
	length, ok := SecretFiles[name]
	if !ok {
		return "", fmt.Errorf("unknown secret: %s", name)
	}

	if length == 0 {
		return "", &ManualSecretError{Filename: name}
	}

	path := filepath.Join(secretsDir, name)

	// Create backup if file exists
	if _, err := os.Stat(path); err == nil {
		// Import time at the top of the file if not already imported
		backupPath := fmt.Sprintf("%s.backup.%d", path, time.Now().Unix())
		if err := os.Rename(path, backupPath); err != nil {
			return "", fmt.Errorf("failed to backup old secret: %w", err)
		}
	}

	secret, err := GenerateRandomString(length)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", name, err)
	}

	return secret, nil
}

// RotateAllSecrets regenerates all auto-generated secrets
// Returns map of filename -> new secret value
// Creates backups of all old secrets
func RotateAllSecrets(secretsDir string) (map[string]string, error) {
	results := make(map[string]string)

	for filename, length := range SecretFiles {
		// Skip user-provided secrets
		if length == 0 {
			continue
		}

		newSecret, err := RotateSecret(secretsDir, filename)
		if err != nil {
			return results, fmt.Errorf("failed to rotate %s: %w", filename, err)
		}
		results[filename] = newSecret
	}

	return results, nil
}

// CleanupBackups removes backup files older than specified duration
func CleanupBackups(secretsDir string, olderThan time.Duration) error {
	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		return fmt.Errorf("failed to read secrets directory: %w", err)
	}

	cutoff := time.Now().Add(-olderThan)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if it's a backup file (ends with .backup.timestamp)
		if strings.Contains(entry.Name(), ".backup.") {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				filePath := filepath.Join(secretsDir, entry.Name())
				if err := os.Remove(filePath); err != nil {
					return fmt.Errorf("failed to remove backup %s: %w", entry.Name(), err)
				}
			}
		}
	}

	return nil
}

// ReadSecret reads a secret from file
// Returns error if file is empty or contains only whitespace
func ReadSecret(secretsDir, name string) (string, error) {
	path := filepath.Join(secretsDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", name, err)
	}

	// Trim whitespace
	secret := strings.TrimSpace(string(data))

	// Check if empty
	if secret == "" {
		return "", &SecretNotConfiguredError{Filename: name}
	}

	return secret, nil
}

// ListSecrets returns all secret files and their status
func ListSecrets(secretsDir string) (map[string]bool, error) {
	result := make(map[string]bool)

	for filename := range SecretFiles {
		path := filepath.Join(secretsDir, filename)
		info, err := os.Stat(path)
		if err != nil {
			result[filename] = false
			continue
		}
		// Check if file has content
		result[filename] = info.Size() > 0
	}

	return result, nil
}
