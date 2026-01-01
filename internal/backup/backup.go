package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Metadata contains information about a backup
type Metadata struct {
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Hostname  string    `json:"hostname"`
	ProjectID string    `json:"project_id"`
	Files     []string  `json:"files"`
}

// Backup represents a backup archive
type Backup struct {
	Name     string
	Path     string
	Metadata Metadata
}

// Manager handles backup operations
type Manager struct {
	projectDir string
	backupDir  string
}

// NewManager creates a new backup manager
func NewManager(projectDir string) *Manager {
	return &Manager{
		projectDir: projectDir,
		backupDir:  filepath.Join(projectDir, "backups"),
	}
}

// Create creates a new backup
func (m *Manager) Create(ctx context.Context) (*Backup, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(m.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup name
	timestamp := time.Now()
	name := fmt.Sprintf("sdbx-backup-%s.tar.gz", timestamp.Format("2006-01-02-150405"))
	backupPath := filepath.Join(m.backupDir, name)

	// Get hostname
	hostname, _ := os.Hostname()

	// Files to backup
	filesToBackup := []string{
		".sdbx.yaml",
		".sdbx.lock",
		"compose.yaml",
		"secrets/",
		"configs/",
	}

	// Create metadata
	metadata := Metadata{
		Version:   "1.0.0",
		Timestamp: timestamp,
		Hostname:  hostname,
		ProjectID: filepath.Base(m.projectDir),
		Files:     filesToBackup,
	}

	// Create tar.gz archive
	if err := m.createArchive(ctx, backupPath, filesToBackup, metadata); err != nil {
		return nil, fmt.Errorf("failed to create archive: %w", err)
	}

	return &Backup{
		Name:     name,
		Path:     backupPath,
		Metadata: metadata,
	}, nil
}

// createArchive creates a tar.gz archive
func (m *Manager) createArchive(ctx context.Context, archivePath string, files []string, metadata Metadata) error {
	// Create archive file
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(f)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Write metadata first
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := m.writeTarEntry(tarWriter, "metadata.json", metadataJSON); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Add each file/directory
	for _, file := range files {
		fullPath := filepath.Join(m.projectDir, file)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// Skip if doesn't exist
			continue
		}

		// Add to archive
		if err := m.addToArchive(ctx, tarWriter, fullPath, file); err != nil {
			return fmt.Errorf("failed to add %s: %w", file, err)
		}
	}

	return nil
}

// addToArchive adds a file or directory to the tar archive
func (m *Manager) addToArchive(ctx context.Context, tw *tar.Writer, fullPath, archivePath string) error {
	// Get file info
	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}

	// If directory, walk it
	if info.IsDir() {
		return filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Get relative path
			relPath, err := filepath.Rel(m.projectDir, path)
			if err != nil {
				return err
			}

			// Skip directories themselves (only files)
			if info.IsDir() {
				return nil
			}

			// Create tar header
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = relPath

			// Write header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			// Write file content
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}

			return nil
		})
	}

	// Single file
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = archivePath

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(tw, file); err != nil {
		return err
	}

	return nil
}

// writeTarEntry writes a byte slice as a tar entry
func (m *Manager) writeTarEntry(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(data)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tw.Write(data); err != nil {
		return err
	}

	return nil
}

// List returns all available backups
func (m *Manager) List(ctx context.Context) ([]*Backup, error) {
	// Check if backup directory exists
	if _, err := os.Stat(m.backupDir); os.IsNotExist(err) {
		return []*Backup{}, nil
	}

	// Read backup directory
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []*Backup
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .tar.gz files
		if filepath.Ext(entry.Name()) != ".gz" {
			continue
		}

		backupPath := filepath.Join(m.backupDir, entry.Name())
		metadata, err := m.readMetadata(backupPath)
		if err != nil {
			// Skip if can't read metadata
			continue
		}

		backups = append(backups, &Backup{
			Name:     entry.Name(),
			Path:     backupPath,
			Metadata: metadata,
		})
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Metadata.Timestamp.After(backups[j].Metadata.Timestamp)
	})

	return backups, nil
}

// readMetadata reads metadata from a backup archive
func (m *Manager) readMetadata(archivePath string) (Metadata, error) {
	var metadata Metadata

	// Open archive
	f, err := os.Open(archivePath)
	if err != nil {
		return metadata, err
	}
	defer f.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return metadata, err
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Read first entry (should be metadata.json)
	header, err := tarReader.Next()
	if err != nil {
		return metadata, err
	}

	if header.Name != "metadata.json" {
		return metadata, fmt.Errorf("first entry is not metadata.json")
	}

	// Read metadata content
	data, err := io.ReadAll(tarReader)
	if err != nil {
		return metadata, err
	}

	// Parse metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return metadata, err
	}

	return metadata, nil
}

// Restore restores a backup
func (m *Manager) Restore(ctx context.Context, backupName string) error {
	backupPath := filepath.Join(m.backupDir, backupName)

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupName)
	}

	// Open archive
	f, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer f.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract all files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Skip metadata.json
		if header.Name == "metadata.json" {
			continue
		}

		// Target path
		targetPath := filepath.Join(m.projectDir, header.Name)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Extract file
		outFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := io.Copy(outFile, tarReader); err != nil {
			outFile.Close()
			return fmt.Errorf("failed to write file: %w", err)
		}

		outFile.Close()

		// Set permissions
		if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	return nil
}

// Delete deletes a backup
func (m *Manager) Delete(ctx context.Context, backupName string) error {
	backupPath := filepath.Join(m.backupDir, backupName)

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupName)
	}

	// Delete file
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// GetSize returns the size of a backup file in bytes
func (b *Backup) GetSize() (int64, error) {
	info, err := os.Stat(b.Path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
