// Copyright (c) 2026 Michael Lechner
// MIT License

package storage

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type Config struct {
    UploadDir         string
    OutputDir         string
    MaxFileSizeMB     int64
    CleanupAfterHours int
    RetentionPolicy   string
}

type Manager struct {
    config Config
}

func NewManager(config Config) (*Manager, error) {
    if err := os.MkdirAll(config.UploadDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create upload dir: %w", err)
    }
    
    if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create output dir: %w", err)
    }
    
    return &Manager{config: config}, nil
}

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

func (m *Manager) GetOutputPath(jobID, originalFilename string) string {
    ext := filepath.Ext(originalFilename)
    if ext == "" {
        ext = ".png"
    }
    
    return filepath.Join(m.config.OutputDir, 
        fmt.Sprintf("%s_upscaled%s", jobID, ext))
}

func (m *Manager) GetOutputDir() string {
    return m.config.OutputDir
}

func (m *Manager) CleanupOldFiles() error {
    cutoff := time.Now().Add(-time.Duration(m.config.CleanupAfterHours) * time.Hour)
    
    for _, dir := range []string{m.config.UploadDir, m.config.OutputDir} {
        if err := m.cleanupDir(dir, cutoff); err != nil {
            return err
        }
    }
    
    return nil
}

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

func sanitizeFilename(filename string) string {
    // Remove path traversal attempts
    filename = filepath.Base(filename)
    return filename
}
