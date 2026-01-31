// Copyright (c) 2026 Michael Lechner
// MIT License

package storage

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
)

// Config holds the configuration settings for the storage manager.
type Config struct {
    UploadDir       string
    OutputDir       string
    MaxFileSizeMB   int64
    CleanupTTL      time.Duration
    RetentionPolicy string
}

// Manager handles file system operations for uploads and outputs.
type Manager struct {
    config Config
}

// NewManager creates a new storage manager and ensures the necessary directories exist.
func NewManager(config Config) (*Manager, error) {
    if err := os.MkdirAll(config.UploadDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create upload dir: %w", err)
    }
    
    if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create output dir: %w", err)
    }
    
    return &Manager{config: config}, nil
}

// SaveUpload saves a byte slice to the upload directory with a unique filename.
// It checks if the file size exceeds the configured maximum.
func (m *Manager) SaveUpload(filename string, data []byte) (string, error) {
    sizeMB := int64(len(data)) / (1024 * 1024)
    if sizeMB > m.config.MaxFileSizeMB {
        return "", fmt.Errorf("file too large: %d MB (max: %d MB)", 
            sizeMB, m.config.MaxFileSizeMB)
    }
    
    path := filepath.Join(m.config.UploadDir, 
        fmt.Sprintf("%d_%s", time.Now().UnixNano(), sanitizeFilename(filename)))
    
    if err := os.WriteFile(path, data, 0644); err != nil {
        return "", fmt.Errorf("failed to save file: %w", err)
    }
    
    return path, nil
}

// GetOutputPath generates the full path for an output file based on the job ID.
func (m *Manager) GetOutputPath(jobID, originalFilename string) string {
    ext := filepath.Ext(originalFilename)
    if ext == "" {
        ext = ".png"
    }
    
    return filepath.Join(m.config.OutputDir, 
        fmt.Sprintf("%s_upscaled%s", jobID, ext))
}

// GetOutputDir returns the configured output directory path.
func (m *Manager) GetOutputDir() string {
    return m.config.OutputDir
}

// CleanupOldFiles removes files in the upload and output directories that are older than the configured retention period.
func (m *Manager) CleanupOldFiles() error {
    cutoff := time.Now().Add(-m.config.CleanupTTL)
    
    for _, dir := range []string{m.config.UploadDir, m.config.OutputDir} {
        if err := m.cleanupDir(dir, cutoff); err != nil {
            return err
        }
    }
    
    return nil
}

// cleanupDir iterates through a directory and removes files older than the cutoff time.
func (m *Manager) cleanupDir(dir string, cutoff time.Time) error {
    entries, err := os.ReadDir(dir)
    if err != nil {
        return fmt.Errorf("failed to read dir %s: %w", dir, err)
    }
    
    for _, entry := range entries {
        info, err := entry.Info()
        if err != nil {
            continue
        }
        
        if info.ModTime().Before(cutoff) {
            path := filepath.Join(dir, entry.Name())
            if err := os.Remove(path); err != nil {
                // Log but continue
                fmt.Printf("Failed to remove %s: %v\n", path, err)
            }
        }
    }
    
    return nil
}

func (m *Manager) DeleteFile(path string) error {
    return os.Remove(path)
}

func (m *Manager) ShouldDeleteAfterDownload() bool {
    return m.config.RetentionPolicy == "delete_after_download"
}

func sanitizeFilename(filename string) string {
    // Remove path traversal attempts
    filename = filepath.Base(filename)
    return filename
}