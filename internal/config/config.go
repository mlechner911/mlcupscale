// Copyright (c) 2026 Michael Lechner
// MIT License

package config

import (
    "fmt"
    "os"
    "time"
    
    "gopkg.in/yaml.v3"
)

type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Upscaler UpscalerConfig `yaml:"upscaler"`
    Storage  StorageConfig  `yaml:"storage"`
    Limits   LimitsConfig   `yaml:"limits"`
    Logging  LoggingConfig  `yaml:"logging"`
    Features FeaturesConfig `yaml:"features"`
}

type ServerConfig struct {
    Host              string `yaml:"host"`
    Port              int    `yaml:"port"`
    ReadTimeout       int    `yaml:"read_timeout_seconds"`
    WriteTimeout      int    `yaml:"write_timeout_seconds"`
    MaxRequestSizeMB  int64  `yaml:"max_request_size_mb"`
}

type UpscalerConfig struct {
    BinaryPath   string `yaml:"binary_path"`
    ModelsPath   string `yaml:"models_path"`
    DefaultModel string `yaml:"default_model"`
    DefaultScale int    `yaml:"default_scale"`
    Threads      string `yaml:"threads"`
    EnableGPU    bool   `yaml:"enable_gpu"`
    GPUID        int    `yaml:"gpu_id"`
}

type StorageConfig struct {
    UploadDir         string `yaml:"upload_dir"`
    OutputDir         string `yaml:"output_dir"`
    MaxFileSizeMB     int64  `yaml:"max_file_size_mb"`
    CleanupAfterHours int    `yaml:"cleanup_after_hours"`
    RetentionPolicy   string `yaml:"retention_policy"`
}

type LimitsConfig struct {
    MaxConcurrentJobs  int `yaml:"max_concurrent_jobs"`
    MaxQueueSize       int `yaml:"max_queue_size"`
    RateLimitPerMinute int `yaml:"rate_limit_per_minute"`
}

type LoggingConfig struct {
    Level    string `yaml:"level"`
    Format   string `yaml:"format"`
    Output   string `yaml:"output"`
    FilePath string `yaml:"file_path"`
}

type FeaturesConfig struct {
    AsyncProcessing bool     `yaml:"async_processing"`
    JobQueue        bool     `yaml:"job_queue"`
    Metrics         bool     `yaml:"metrics"`
    CORSEnabled     bool     `yaml:"cors_enabled"`
    AllowedOrigins  []string `yaml:"allowed_origins"`
}

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }
    
    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    // Apply environment variable overrides
    applyEnvOverrides(&config)
    
    // Validate
    if err := validate(&config); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    return &config, nil
}

func applyEnvOverrides(cfg *Config) {
    if port := os.Getenv("UPSCALE_SERVER_PORT"); port != "" {
        fmt.Sscanf(port, "%d", &cfg.Server.Port)
    }
    if host := os.Getenv("UPSCALE_SERVER_HOST"); host != "" {
        cfg.Server.Host = host
    }
    if binary := os.Getenv("UPSCALE_BINARY_PATH"); binary != "" {
        cfg.Upscaler.BinaryPath = binary
    }
    if models := os.Getenv("UPSCALE_MODELS_PATH"); models != "" {
        cfg.Upscaler.ModelsPath = models
    }
    if threads := os.Getenv("UPSCALE_THREADS"); threads != "" {
        cfg.Upscaler.Threads = threads
    }
    if level := os.Getenv("UPSCALE_LOG_LEVEL"); level != "" {
        cfg.Logging.Level = level
    }
}

func validate(cfg *Config) error {
    if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
        return fmt.Errorf("invalid port: %d", cfg.Server.Port)
    }
    
    // We don't check for binary/models existence here anymore because 
    // it might not be present during initial startup/docker build
    
    return nil
}

func (c *Config) GetReadTimeout() time.Duration {
    return time.Duration(c.Server.ReadTimeout) * time.Second
}

func (c *Config) GetWriteTimeout() time.Duration {
    return time.Duration(c.Server.WriteTimeout) * time.Second
}
