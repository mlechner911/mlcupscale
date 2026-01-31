// Copyright (c) 2026 Michael Lechner
// MIT License

package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level configuration structure for the application.
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Upscaler UpscalerConfig `yaml:"upscaler"`
    Storage  StorageConfig  `yaml:"storage"`
    Limits   LimitsConfig   `yaml:"limits"`
    Logging  LoggingConfig  `yaml:"logging"`
    Features FeaturesConfig `yaml:"features"`
}

// ServerConfig holds the HTTP server settings.
type ServerConfig struct {
    Host              string `yaml:"host"`
    Port              int    `yaml:"port"`
    APIPrefix         string `yaml:"api_prefix"`
    AuthToken         string `yaml:"auth_token"`
    ReadTimeout       int    `yaml:"read_timeout_seconds"`
    WriteTimeout      int    `yaml:"write_timeout_seconds"`
    MaxRequestSizeMB  int64  `yaml:"max_request_size_mb"`
}

// UpscalerConfig holds the settings for the upscaling engine.
type UpscalerConfig struct {
    BinaryPath   string `yaml:"binary_path"`
    ModelsPath   string `yaml:"models_path"`
    DefaultModel string `yaml:"default_model"`
    DefaultScale int    `yaml:"default_scale"`
    Threads      string `yaml:"threads"`
    EnableGPU    bool   `yaml:"enable_gpu"`
    GPUID        int    `yaml:"gpu_id"`
}

// StorageConfig holds settings for file storage locations and cleanup policies.
type StorageConfig struct {
    UploadDir         string `yaml:"upload_dir"`
    OutputDir         string `yaml:"output_dir"`
    MaxFileSizeMB     int64  `yaml:"max_file_size_mb"`
    CleanupAfterHours int    `yaml:"cleanup_after_hours"`
    RetentionPolicy   string `yaml:"retention_policy"`
}

// LimitsConfig holds concurrency and rate limiting settings.
type LimitsConfig struct {
    MaxConcurrentJobs  int `yaml:"max_concurrent_jobs"`
    MaxQueueSize       int `yaml:"max_queue_size"`
    RateLimitPerMinute int `yaml:"rate_limit_per_minute"`
}

// LoggingConfig holds logging preferences.
type LoggingConfig struct {
    Level    string `yaml:"level"`
    Format   string `yaml:"format"`
    Output   string `yaml:"output"`
    FilePath string `yaml:"file_path"`
}

// FeaturesConfig holds flags to enable or disable specific application features.
type FeaturesConfig struct {
    AsyncProcessing bool     `yaml:"async_processing"`
    JobQueue        bool     `yaml:"job_queue"`
    Metrics         bool     `yaml:"metrics"`
    CORSEnabled     bool     `yaml:"cors_enabled"`
    EnableSwagger   bool     `yaml:"enable_swagger"`
    EnableWebUI     bool     `yaml:"enable_web_ui"`
    AllowedOrigins  []string `yaml:"allowed_origins"`
}

// Load reads the configuration from a YAML file and applies environment variable overrides.
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

    // Set default prefix if empty
    if config.Server.APIPrefix == "" {
        config.Server.APIPrefix = "/api/v1"
    }

    // Validate
    if err := validate(&config); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return &config, nil
}

// applyEnvOverrides checks for specific environment variables and updates the configuration if present.
func applyEnvOverrides(cfg *Config) {
    if port := os.Getenv("UPSCALE_SERVER_PORT"); port != "" {
        fmt.Sscanf(port, "%d", &cfg.Server.Port)
    }
    if host := os.Getenv("UPSCALE_SERVER_HOST"); host != "" {
        cfg.Server.Host = host
    }
    if prefix := os.Getenv("UPSCALE_API_PREFIX"); prefix != "" {
        cfg.Server.APIPrefix = prefix
    }
    if auth := os.Getenv("UPSCALE_AUTH_TOKEN"); auth != "" {
        cfg.Server.AuthToken = auth
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
    if uploadDir := os.Getenv("UPSCALE_STORAGE_UPLOAD_DIR"); uploadDir != "" {
        cfg.Storage.UploadDir = uploadDir
    }
    if outputDir := os.Getenv("UPSCALE_STORAGE_OUTPUT_DIR"); outputDir != "" {
        cfg.Storage.OutputDir = outputDir
    }
}


// validate performs basic sanity checks on the configuration values.
func validate(cfg *Config) error {
    if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
        return fmt.Errorf("invalid port: %d", cfg.Server.Port)
    }
    return nil
}

// GetReadTimeout converts the configured read timeout to a time.Duration.
func (c *Config) GetReadTimeout() time.Duration {
    return time.Duration(c.Server.ReadTimeout) * time.Second
}

// GetWriteTimeout converts the configured write timeout to a time.Duration.
func (c *Config) GetWriteTimeout() time.Duration {
    return time.Duration(c.Server.WriteTimeout) * time.Second
}
