# Image Upscale Service - Detaillierter Projektplan


## Projekt-Übersicht

**Name**: upscale-service  (mlc-upscaler v1.0)
**Zweck**: HTTP REST API für AI-basiertes Image Upscaling mit Real-ESRGAN  
**Technologie**: Go (Gin Framework) + Real-ESRGAN-ncnn-vulkan  
**Deployment**: Primär Docker, optional als systemd/launchd/Windows Service  

## Zielsetzung

1. **Mac Studio**: Generiert 1600x1200 Bilder mit Ollama Vision
2. **Server**: Upscaled Bilder auf 6400x4800 (Poster-Größe)
3. **Integration**: Einfache HTTP API für automatisierte Workflows
4. **Deployment**: Docker-first, aber auch als native Service lauffähig

---

## 1. Projekt-Struktur

```
upscale-service/
├── cmd/
│   ├── server/
│   │   └── main.go                 # Server-Einstiegspunkt
│   └── client/
│       └── main.go                 # CLI-Client (optional)
│
├── internal/
│   ├── api/
│   │   ├── handlers.go             # Gin HTTP Handlers
│   │   ├── middleware.go           # Logging, CORS, Rate-Limiting
│   │   └── routes.go               # Route-Definitionen
│   ├── upscaler/
│   │   ├── upscaler.go             # Upscaling-Core-Logik
│   │   ├── models.go               # Modell-Management
│   │   └── upscaler_test.go        # Unit-Tests
│   ├── storage/
│   │   ├── storage.go              # File-Management
│   │   └── cleanup.go              # Automatisches Cleanup
│   └── config/
│       └── config.go               # Konfigurations-Handling
│
├── bin/                             # ncnn binaries (wird beim Build kopiert)
│   ├── realesrgan-ncnn-vulkan-linux
│   ├── realesrgan-ncnn-vulkan-darwin
│   └── realesrgan-ncnn-vulkan.exe
│
├── models/                          # AI-Modelle
│   ├── realesrgan-x4plus/
│   ├── realesrgan-x4plus-anime/
│   └── realesr-animevideov3/
│
├── config/
│   ├── config.yaml                 # Hauptkonfiguration
│   ├── config.docker.yaml          # Docker-spezifisch
│   └── config.dev.yaml             # Development
│
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile              # Multi-stage Build
│   │   ├── Dockerfile.gpu          # Mit NVIDIA GPU Support
│   │   └── docker-compose.yml      # Compose-Setup
│   ├── systemd/
│   │   └── upscale-service.service # Linux systemd
│   ├── launchd/
│   │   └── com.upscale.service.plist # macOS launchd
│   └── windows/
│       └── install-service.ps1     # Windows Service
│
├── scripts/
│   ├── build.sh                    # Build-Script
│   ├── download-models.sh          # Modelle herunterladen
│   ├── build-docker.sh             # Docker Image bauen
│   └── release.sh                  # Release-Packaging
│
├── test/
│   ├── integration/
│   │   └── api_test.go             # Integration Tests
│   └── testdata/
│       └── sample_images/          # Test-Bilder
│
├── docs/
│   ├── API.md                      # API-Dokumentation
│   ├── DEPLOYMENT.md               # Deployment-Guide
│   └── DEVELOPMENT.md              # Development-Setup
│
├── .github/
│   └── workflows/
│       ├── build.yml               # CI/CD Pipeline
│       └── release.yml             # Release Automation
│
├── go.mod
├── go.sum
├── Makefile
├── .dockerignore
├── .gitignore
├── LICENSE
└── README.md
```

---

## 2. Technologie-Stack

### Backend
- **Sprache**: Go 1.21+
- **Web-Framework**: Gin (https://gin-gonic.com)
- **Config**: YAML via gopkg.in/yaml.v3
- **Logging**: zerolog oder zap

### AI-Engine
- **Framework**: ncnn (Tencent Neural Network)
- **Modell**: Real-ESRGAN (RRDB-basiert)
- **Acceleration**: Vulkan (CPU + GPU)

### Deployment
- **Container**: Docker / Docker Compose
- **Orchestration**: Optional Kubernetes
- **Service Management**: systemd (Linux), launchd (macOS), Windows Service

---

## 3. API-Spezifikation

### Endpoints

#### POST /api/v1/upscale
Upscale ein Bild

**Request:**
```http
POST /api/v1/upscale
Content-Type: multipart/form-data

image: [binary]
scale: 4
model_name: realesrgan-x4plus
tile_size: 0 (optional)
format: png (optional: png, jpg, webp)
```

**Response (200 OK):**
```json
{
  "success": true,
  "job_id": "abc123...",
  "download_url": "/api/v1/download/abc123",
  "duration_seconds": 45.2,
  "input_size": {
    "width": 1600,
    "height": 1200
  },
  "output_size": {
    "width": 6400,
    "height": 4800
  },
  "file_size_bytes": 27458921
}
```

**Response (400 Bad Request):**
```json
{
  "success": false,
  "error": "invalid scale: must be 2, 3, or 4"
}
```

#### GET /api/v1/download/:job_id
Download upscaled Bild

**Response:**
- Binary image data mit Content-Type Header

#### GET /api/v1/models
Liste verfügbare Modelle

**Response:**
```json
{
  "models": [
    {
      "name": "realesrgan-x4plus",
      "description": "General purpose 4x upscaling",
      "supported_scales": [2, 3, 4]
    },
    {
      "name": "realesrgan-x4plus-anime",
      "description": "Optimized for anime/illustrations",
      "supported_scales": [4]
    }
  ]
}
```

#### GET /api/v1/health
Health-Check

**Response:**
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime_seconds": 12345,
  "models_loaded": 3
}
```

#### GET /api/v1/status/:job_id
Job-Status (für async Processing)

**Response:**
```json
{
  "job_id": "abc123",
  "status": "processing",
  "progress_percent": 45,
  "estimated_seconds_remaining": 30
}
```

---

## 4. Konfiguration

### config/config.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout_seconds: 300
  write_timeout_seconds: 300
  max_request_size_mb: 100
  
upscaler:
  binary_path: "./bin/realesrgan-ncnn-vulkan"
  models_path: "./models"
  default_model: "realesrgan-x4plus"
  default_scale: 4
  threads: "12:12:12"
  enable_gpu: true
  gpu_id: -1  # -1 = auto-detect
  
storage:
  upload_dir: "./data/uploads"
  output_dir: "./data/outputs"
  max_file_size_mb: 100
  cleanup_after_hours: 24
  retention_policy: "delete_after_download"  # or "keep"
  
limits:
  max_concurrent_jobs: 4
  max_queue_size: 20
  rate_limit_per_minute: 10
  
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json or text
  output: "stdout"  # stdout, file, or both
  file_path: "./logs/upscale-service.log"
  
features:
  async_processing: true
  job_queue: true
  metrics: true
  cors_enabled: true
  allowed_origins:
    - "http://localhost:3000"
    - "http://macstudio.local"
```

### Environment Variables (Override)

```bash
# Server
UPSCALE_SERVER_PORT=8080
UPSCALE_SERVER_HOST=0.0.0.0

# Paths
UPSCALE_BINARY_PATH=/app/bin/realesrgan-ncnn-vulkan
UPSCALE_MODELS_PATH=/app/models

# Performance
UPSCALE_THREADS=12:12:12
UPSCALE_MAX_CONCURRENT_JOBS=4

# Storage
UPSCALE_UPLOAD_DIR=/data/uploads
UPSCALE_OUTPUT_DIR=/data/outputs

# Logging
UPSCALE_LOG_LEVEL=info
```

---

## 5. Core-Komponenten Implementation

### 5.1 Config Loader (internal/config/config.go)

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"
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
    
    if _, err := os.Stat(cfg.Upscaler.BinaryPath); os.IsNotExist(err) {
        return fmt.Errorf("binary not found: %s", cfg.Upscaler.BinaryPath)
    }
    
    if _, err := os.Stat(cfg.Upscaler.ModelsPath); os.IsNotExist(err) {
        return fmt.Errorf("models path not found: %s", cfg.Upscaler.ModelsPath)
    }
    
    return nil
}

func (c *Config) GetReadTimeout() time.Duration {
    return time.Duration(c.Server.ReadTimeout) * time.Second
}

func (c *Config) GetWriteTimeout() time.Duration {
    return time.Duration(c.Server.WriteTimeout) * time.Second
}
```

### 5.2 Upscaler Service (internal/upscaler/upscaler.go)

```go
package upscaler

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "sync"
    "time"
)

type Config struct {
    BinaryPath   string
    ModelsPath   string
    DefaultModel string
    DefaultScale int
    Threads      string
    EnableGPU    bool
    GPUID        int
}

type Request struct {
    InputPath  string
    OutputPath string
    Scale      int
    ModelName  string
    TileSize   int
    Format     string
}

type Result struct {
    OutputPath string
    Duration   time.Duration
    InputSize  ImageSize
    OutputSize ImageSize
    FileSizeBytes int64
}

type ImageSize struct {
    Width  int `json:"width"`
    Height int `json:"height"`
}

type Job struct {
    ID        string
    Request   Request
    Status    string
    Progress  int
    StartTime time.Time
    Result    *Result
    Error     error
}

type Service struct {
    config Config
    jobs   map[string]*Job
    jobsMu sync.RWMutex
    queue  chan *Job
}

func NewService(config Config) *Service {
    return &Service{
        config: config,
        jobs:   make(map[string]*Job),
        queue:  make(chan *Job, 100),
    }
}

func (s *Service) StartWorkers(numWorkers int) {
    for i := 0; i < numWorkers; i++ {
        go s.worker()
    }
}

func (s *Service) worker() {
    for job := range s.queue {
        s.processJob(job)
    }
}

func (s *Service) SubmitJob(req Request) (string, error) {
    jobID := generateJobID()
    
    job := &Job{
        ID:        jobID,
        Request:   req,
        Status:    "queued",
        StartTime: time.Now(),
    }
    
    s.jobsMu.Lock()
    s.jobs[jobID] = job
    s.jobsMu.Unlock()
    
    s.queue <- job
    
    return jobID, nil
}

func (s *Service) GetJob(jobID string) (*Job, bool) {
    s.jobsMu.RLock()
    defer s.jobsMu.RUnlock()
    job, ok := s.jobs[jobID]
    return job, ok
}

func (s *Service) processJob(job *Job) {
    job.Status = "processing"
    
    result, err := s.Upscale(context.Background(), job.Request)
    
    s.jobsMu.Lock()
    defer s.jobsMu.Unlock()
    
    if err != nil {
        job.Status = "failed"
        job.Error = err
    } else {
        job.Status = "completed"
        job.Result = result
    }
}

func (s *Service) Upscale(ctx context.Context, req Request) (*Result, error) {
    start := time.Now()
    
    // Validate
    if err := s.validate(req); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    // Get input size
    inputSize, err := s.getImageSize(req.InputPath)
    if err != nil {
        return nil, fmt.Errorf("failed to get input size: %w", err)
    }
    
    // Build command
    args := s.buildArgs(req)
    
    // Execute
    cmd := exec.CommandContext(ctx, s.config.BinaryPath, args...)
    if output, err := cmd.CombinedOutput(); err != nil {
        return nil, fmt.Errorf("upscale failed: %w\nOutput: %s", err, output)
    }
    
    // Get output size
    outputSize, err := s.getImageSize(req.OutputPath)
    if err != nil {
        return nil, fmt.Errorf("failed to get output size: %w", err)
    }
    
    // Get file size
    stat, err := os.Stat(req.OutputPath)
    if err != nil {
        return nil, fmt.Errorf("failed to stat output: %w", err)
    }
    
    return &Result{
        OutputPath:    req.OutputPath,
        Duration:      time.Since(start),
        InputSize:     inputSize,
        OutputSize:    outputSize,
        FileSizeBytes: stat.Size(),
    }, nil
}

func (s *Service) validate(req Request) error {
    if _, err := os.Stat(s.config.BinaryPath); os.IsNotExist(err) {
        return fmt.Errorf("upscaler binary not found: %s", s.config.BinaryPath)
    }
    
    if _, err := os.Stat(req.InputPath); os.IsNotExist(err) {
        return fmt.Errorf("input file not found: %s", req.InputPath)
    }
    
    if req.Scale < 2 || req.Scale > 4 {
        return fmt.Errorf("invalid scale: %d (must be 2, 3, or 4)", req.Scale)
    }
    
    modelPath := filepath.Join(s.config.ModelsPath, req.ModelName)
    if _, err := os.Stat(modelPath); os.IsNotExist(err) {
        return fmt.Errorf("model not found: %s", req.ModelName)
    }
    
    return nil
}

func (s *Service) buildArgs(req Request) []string {
    args := []string{
        "-i", req.InputPath,
        "-o", req.OutputPath,
        "-s", fmt.Sprintf("%d", req.Scale),
        "-m", s.config.ModelsPath,
        "-n", req.ModelName,
        "-j", s.config.Threads,
    }
    
    if req.TileSize > 0 {
        args = append(args, "-t", fmt.Sprintf("%d", req.TileSize))
    }
    
    if !s.config.EnableGPU {
        args = append(args, "-g", "-1")
    } else if s.config.GPUID >= 0 {
        args = append(args, "-g", fmt.Sprintf("%d", s.config.GPUID))
    }
    
    if req.Format != "" {
        args = append(args, "-f", req.Format)
    }
    
    return args
}

func (s *Service) getImageSize(path string) (ImageSize, error) {
    cmd := exec.Command("identify", "-format", "%wx%h", path)
    output, err := cmd.Output()
    if err != nil {
        return ImageSize{}, fmt.Errorf("identify failed: %w", err)
    }
    
    var width, height int
    if _, err := fmt.Sscanf(string(output), "%dx%d", &width, &height); err != nil {
        return ImageSize{}, fmt.Errorf("failed to parse size: %w", err)
    }
    
    return ImageSize{Width: width, Height: height}, nil
}

func (s *Service) GetAvailableModels() ([]ModelInfo, error) {
    entries, err := os.ReadDir(s.config.ModelsPath)
    if err != nil {
        return nil, err
    }
    
    models := make([]ModelInfo, 0)
    for _, entry := range entries {
        if entry.IsDir() {
            info := ModelInfo{
                Name:            entry.Name(),
                SupportedScales: []int{2, 3, 4},
            }
            
            switch entry.Name() {
            case "realesrgan-x4plus":
                info.Description = "General purpose 4x upscaling for photos"
            case "realesrgan-x4plus-anime":
                info.Description = "Optimized for anime and illustrations"
                info.SupportedScales = []int{4}
            case "realesr-animevideov3":
                info.Description = "Anime/video optimized with 2x/3x/4x support"
            default:
                info.Description = "Custom model"
            }
            
            models = append(models, info)
        }
    }
    
    return models, nil
}

type ModelInfo struct {
    Name            string `json:"name"`
    Description     string `json:"description"`
    SupportedScales []int  `json:"supported_scales"`
}

func generateJobID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
```

### 5.3 Storage Manager (internal/storage/storage.go)

```go
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
    
    // TODO: Add more sanitization if needed
    
    return filename
}
```

### 5.4 API Handlers (internal/api/handlers.go)

```go
package api

import (
    "context"
    "fmt"
    "net/http"
    "path/filepath"
    "time"
    
    "github.com/gin-gonic/gin"
    
    "github.com/yourusername/upscale-service/internal/storage"
    "github.com/yourusername/upscale-service/internal/upscaler"
)

type Handler struct {
    upscaler *upscaler.Service
    storage  *storage.Manager
}

func NewHandler(upscalerService *upscaler.Service, storageManager *storage.Manager) *Handler {
    return &Handler{
        upscaler: upscalerService,
        storage:  storageManager,
    }
}

type UpscaleRequest struct {
    Scale     int    `form:"scale" json:"scale"`
    ModelName string `form:"model_name" json:"model_name"`
    TileSize  int    `form:"tile_size" json:"tile_size"`
    Format    string `form:"format" json:"format"`
}

type UpscaleResponse struct {
    Success       bool                  `json:"success"`
    JobID         string                `json:"job_id,omitempty"`
    DownloadURL   string                `json:"download_url,omitempty"`
    Duration      float64               `json:"duration_seconds,omitempty"`
    InputSize     *upscaler.ImageSize   `json:"input_size,omitempty"`
    OutputSize    *upscaler.ImageSize   `json:"output_size,omitempty"`
    FileSizeBytes int64                 `json:"file_size_bytes,omitempty"`
    Error         string                `json:"error,omitempty"`
}

func (h *Handler) HandleUpscale(c *gin.Context) {
    var req UpscaleRequest
    if err := c.ShouldBind(&req); err != nil {
        c.JSON(http.StatusBadRequest, UpscaleResponse{
            Success: false,
            Error:   fmt.Sprintf("invalid request: %v", err),
        })
        return
    }
    
    fileHeader, err := c.FormFile("image")
    if err != nil {
        c.JSON(http.StatusBadRequest, UpscaleResponse{
            Success: false,
            Error:   "no image provided",
        })
        return
    }
    
    file, err := fileHeader.Open()
    if err != nil {
        c.JSON(http.StatusInternalServerError, UpscaleResponse{
            Success: false,
            Error:   "failed to read file",
        })
        return
    }
    defer file.Close()
    
    data := make([]byte, fileHeader.Size)
    if _, err := file.Read(data); err != nil {
        c.JSON(http.StatusInternalServerError, UpscaleResponse{
            Success: false,
            Error:   "failed to read file data",
        })
        return
    }
    
    inputPath, err := h.storage.SaveUpload(fileHeader.Filename, data)
    if err != nil {
        c.JSON(http.StatusInternalServerError, UpscaleResponse{
            Success: false,
            Error:   fmt.Sprintf("failed to save upload: %v", err),
        })
        return
    }
    
    jobID, err := h.upscaler.SubmitJob(upscaler.Request{
        InputPath:  inputPath,
        OutputPath: h.storage.GetOutputPath(jobID, fileHeader.Filename),
        Scale:      req.Scale,
        ModelName:  req.ModelName,
        TileSize:   req.TileSize,
        Format:     req.Format,
    })
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, UpscaleResponse{
            Success: false,
            Error:   fmt.Sprintf("failed to submit job: %v", err),
        })
        return
    }
    
    // Wait for completion or return immediately for async
    job, _ := h.upscaler.GetJob(jobID)
    
    // For now, wait for completion
    timeout := time.After(10 * time.Minute)
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-timeout:
            c.JSON(http.StatusRequestTimeout, UpscaleResponse{
                Success: false,
                Error:   "processing timeout",
            })
            return
        case <-ticker.C:
            job, ok := h.upscaler.GetJob(jobID)
            if !ok {
                c.JSON(http.StatusInternalServerError, UpscaleResponse{
                    Success: false,
                    Error:   "job not found",
                })
                return
            }
            
            if job.Status == "completed" {
                c.JSON(http.StatusOK, UpscaleResponse{
                    Success:       true,
                    JobID:         jobID,
                    DownloadURL:   "/api/v1/download/" + jobID,
                    Duration:      job.Result.Duration.Seconds(),
                    InputSize:     &job.Result.InputSize,
                    OutputSize:    &job.Result.OutputSize,
                    FileSizeBytes: job.Result.FileSizeBytes,
                })
                return
            }
            
            if job.Status == "failed" {
                c.JSON(http.StatusInternalServerError, UpscaleResponse{
                    Success: false,
                    Error:   job.Error.Error(),
                })
                return
            }
        }
    }
}

func (h *Handler) HandleDownload(c *gin.Context) {
    jobID := c.Param("job_id")
    
    job, ok := h.upscaler.GetJob(jobID)
    if !ok {
        c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
        return
    }
    
    if job.Status != "completed" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "job not completed",
            "status": job.Status,
        })
        return
    }
    
    c.File(job.Result.OutputPath)
    
    // Optional: Delete after download based on retention policy
    // h.storage.DeleteFile(job.Result.OutputPath)
}

func (h *Handler) HandleStatus(c *gin.Context) {
    jobID := c.Param("job_id")
    
    job, ok := h.upscaler.GetJob(jobID)
    if !ok {
        c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "job_id":   job.ID,
        "status":   job.Status,
        "progress": job.Progress,
    })
}

func (h *Handler) HandleModels(c *gin.Context) {
    models, err := h.upscaler.GetAvailableModels()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": fmt.Sprintf("failed to list models: %v", err),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "models": models,
    })
}

func (h *Handler) HandleHealth(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":  "ok",
        "version": "1.0.0",
        "time":    time.Now().Unix(),
    })
}
```

### 5.5 Main Server (cmd/server/main.go)

```go
package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "time"
    
    "github.com/gin-gonic/gin"
    
    "github.com/yourusername/upscale-service/internal/api"
    "github.com/yourusername/upscale-service/internal/config"
    "github.com/yourusername/upscale-service/internal/storage"
    "github.com/yourusername/upscale-service/internal/upscaler"
)

var (
    configPath = flag.String("config", "config/config.yaml", "Path to config file")
    version    = "1.0.0"
)

func main() {
    flag.Parse()
    
    log.Printf("Starting Upscale Service v%s", version)
    
    // Load config
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Make paths absolute if relative
    if !filepath.IsAbs(cfg.Upscaler.BinaryPath) {
        cfg.Upscaler.BinaryPath = filepath.Join(getBaseDir(), cfg.Upscaler.BinaryPath)
    }
    if !filepath.IsAbs(cfg.Upscaler.ModelsPath) {
        cfg.Upscaler.ModelsPath = filepath.Join(getBaseDir(), cfg.Upscaler.ModelsPath)
    }
    if !filepath.IsAbs(cfg.Storage.UploadDir) {
        cfg.Storage.UploadDir = filepath.Join(getBaseDir(), cfg.Storage.UploadDir)
    }
    if !filepath.IsAbs(cfg.Storage.OutputDir) {
        cfg.Storage.OutputDir = filepath.Join(getBaseDir(), cfg.Storage.OutputDir)
    }
    
    // Initialize services
    upscalerService := upscaler.NewService(upscaler.Config{
        BinaryPath:   cfg.Upscaler.BinaryPath,
        ModelsPath:   cfg.Upscaler.ModelsPath,
        DefaultModel: cfg.Upscaler.DefaultModel,
        DefaultScale: cfg.Upscaler.DefaultScale,
        Threads:      cfg.Upscaler.Threads,
        EnableGPU:    cfg.Upscaler.EnableGPU,
        GPUID:        cfg.Upscaler.GPUID,
    })
    
    // Start workers
    upscalerService.StartWorkers(cfg.Limits.MaxConcurrentJobs)
    
    storageManager, err := storage.NewManager(storage.Config{
        UploadDir:         cfg.Storage.UploadDir,
        OutputDir:         cfg.Storage.OutputDir,
        MaxFileSizeMB:     cfg.Storage.MaxFileSizeMB,
        CleanupAfterHours: cfg.Storage.CleanupAfterHours,
        RetentionPolicy:   cfg.Storage.RetentionPolicy,
    })
    if err != nil {
        log.Fatalf("Failed to initialize storage: %v", err)
    }
    
    // Start cleanup routine
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()
        
        for range ticker.C {
            if err := storageManager.CleanupOldFiles(); err != nil {
                log.Printf("Cleanup failed: %v", err)
            }
        }
    }()
    
    // Setup API
    handler := api.NewHandler(upscalerService, storageManager)
    
    if cfg.Logging.Level == "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    router := gin.Default()
    
    // Middleware
    router.Use(gin.Recovery())
    
    if cfg.Features.CORSEnabled {
        router.Use(corsMiddleware(cfg.Features.AllowedOrigins))
    }
    
    // Routes
    v1 := router.Group("/api/v1")
    {
        v1.POST("/upscale", handler.HandleUpscale)
        v1.GET("/download/:job_id", handler.HandleDownload)
        v1.GET("/status/:job_id", handler.HandleStatus)
        v1.GET("/models", handler.HandleModels)
        v1.GET("/health", handler.HandleHealth)
    }
    
    // Start server
    addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    log.Printf("Server listening on %s", addr)
    
    if err := router.Run(addr); err != nil {
        log.Fatal(err)
    }
}

func getBaseDir() string {
    exe, err := os.Executable()
    if err != nil {
        return "."
    }
    return filepath.Dir(exe)
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
    return func(c *gin.Context) {
        origin := c.Request.Header.Get("Origin")
        
        allowed := false
        for _, allowedOrigin := range allowedOrigins {
            if origin == allowedOrigin || allowedOrigin == "*" {
                allowed = true
                break
            }
        }
        
        if allowed {
            c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
            c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
            c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        }
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    }
}
```

---

## 6. Docker Implementation

### 6.1 Dockerfile (Multi-Stage Build)

```dockerfile
# Stage 1: Build ncnn
FROM ubuntu:24.04 AS ncnn-builder

RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    libvulkan-dev \
    libwebp-dev \
    glslang-tools \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Clone Real-ESRGAN-ncnn-vulkan
RUN git clone --recursive https://github.com/xinntao/Real-ESRGAN-ncnn-vulkan.git

WORKDIR /build/Real-ESRGAN-ncnn-vulkan/src

# Build with optimizations
RUN cmake -B build \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CXX_FLAGS="-O3" \
    -DCMAKE_C_FLAGS="-O3" && \
    cmake --build build -j$(nproc)

# Stage 2: Build Go application
FROM golang:1.21-alpine AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o upscale-server \
    ./cmd/server

# Stage 3: Runtime
FROM ubuntu:24.04

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libvulkan1 \
    mesa-vulkan-drivers \
    libwebp7 \
    imagemagick \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create app user
RUN useradd -m -u 1000 appuser

WORKDIR /app

# Copy binaries
COPY --from=ncnn-builder /build/Real-ESRGAN-ncnn-vulkan/src/build/realesrgan-ncnn-vulkan ./bin/
COPY --from=go-builder /app/upscale-server ./

# Copy models (will be downloaded separately)
COPY models ./models/
COPY config ./config/

# Create data directories
RUN mkdir -p data/uploads data/outputs && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

CMD ["./upscale-server", "-config", "config/config.docker.yaml"]
```

### 6.2 Docker Compose

```yaml
version: '3.8'

services:
  upscale-service:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile
    image: upscale-service:latest
    container_name: upscale-service
    ports:
      - "8080:8080"
    volumes:
      - ./data/uploads:/app/data/uploads
      - ./data/outputs:/app/data/outputs
      - ./models:/app/models:ro
    environment:
      - UPSCALE_LOG_LEVEL=info
      - UPSCALE_THREADS=12:12:12
      - UPSCALE_MAX_CONCURRENT_JOBS=4
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 3s
      retries: 3
    deploy:
      resources:
        limits:
          memory: 16G
        reservations:
          memory: 4G
```

### 6.3 GPU-enabled Dockerfile

```dockerfile
FROM nvidia/cuda:12.0.0-base-ubuntu22.04

# ... (similar to above but with CUDA support)

# Install CUDA Vulkan ICD
RUN apt-get update && apt-get install -y \
    nvidia-vulkan-icd \
    && rm -rf /var/lib/apt/lists/*

# Rest same as above...
```

---

## 7. Build & Deployment Scripts

### 7.1 Build Script (scripts/build.sh)

```bash
#!/bin/bash
set -e

echo "=== Building Upscale Service ==="

VERSION=${1:-"1.0.0"}
BUILD_DIR="build"
DIST_DIR="dist/upscale-service-$VERSION"

# Clean
rm -rf "$BUILD_DIR" "$DIST_DIR"
mkdir -p "$BUILD_DIR" "$DIST_DIR"

echo "Building for multiple platforms..."

# Linux
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$BUILD_DIR/upscale-server-linux-amd64" \
    ./cmd/server

# macOS (Intel)
echo "Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$BUILD_DIR/upscale-server-darwin-amd64" \
    ./cmd/server

# macOS (Apple Silicon)
echo "Building for macOS (ARM)..."
GOOS=darwin GOARCH=arm64 go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$BUILD_DIR/upscale-server-darwin-arm64" \
    ./cmd/server

# Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$BUILD_DIR/upscale-server-windows-amd64.exe" \
    ./cmd/server

echo "=== Build complete ==="
ls -lh "$BUILD_DIR"
```

### 7.2 Model Download Script (scripts/download-models.sh)

```bash
#!/bin/bash
set -e

MODELS_DIR="models"
TEMP_DIR="/tmp/realesrgan-models"

echo "=== Downloading Real-ESRGAN Models ==="

mkdir -p "$MODELS_DIR"
mkdir -p "$TEMP_DIR"

cd "$TEMP_DIR"

# Download model pack
echo "Downloading model pack..."
wget -O models.zip \
    https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesrgan-ncnn-vulkan-20220424-ubuntu.zip

# Extract
echo "Extracting..."
unzip -q models.zip

# Copy models
echo "Copying models..."
cp -r realesrgan-ncnn-vulkan-20220424-ubuntu/models/* "$MODELS_DIR/"

# Cleanup
cd -
rm -rf "$TEMP_DIR"

echo "=== Models installed ==="
ls -lh "$MODELS_DIR"
```

### 7.3 Docker Build Script (scripts/build-docker.sh)

```bash
#!/bin/bash
set -e

VERSION=${1:-"latest"}
IMAGE_NAME="upscale-service"

echo "=== Building Docker Image ==="

# Download models if not present
if [ ! -d "models/realesrgan-x4plus" ]; then
    echo "Models not found, downloading..."
    ./scripts/download-models.sh
fi

# Build image
echo "Building Docker image..."
docker build \
    -f deployments/docker/Dockerfile \
    -t "$IMAGE_NAME:$VERSION" \
    -t "$IMAGE_NAME:latest" \
    .

echo "=== Docker image built: $IMAGE_NAME:$VERSION ==="

# Optional: Run tests
echo "Testing image..."
docker run --rm "$IMAGE_NAME:$VERSION" ./upscale-server -help

echo "=== Build complete ==="
```

---

## 8. Service Installation

### 8.1 Linux systemd (deployments/systemd/upscale-service.service)

```ini
[Unit]
Description=Image Upscale Service
After=network.target

[Service]
Type=simple
User=upscale
Group=upscale
WorkingDirectory=/opt/upscale-service
ExecStart=/opt/upscale-service/upscale-server -config /opt/upscale-service/config/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=upscale-service

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/upscale-service/data

[Install]
WantedBy=multi-user.target
```

**Installation:**
```bash
sudo useradd -r -s /bin/false upscale
sudo mkdir -p /opt/upscale-service
sudo cp -r build/* /opt/upscale-service/
sudo chown -R upscale:upscale /opt/upscale-service
sudo cp deployments/systemd/upscale-service.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable upscale-service
sudo systemctl start upscale-service
```

### 8.2 macOS launchd (deployments/launchd/com.upscale.service.plist)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.upscale.service</string>
    
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/opt/upscale-service/upscale-server</string>
        <string>-config</string>
        <string>/usr/local/opt/upscale-service/config/config.yaml</string>
    </array>
    
    <key>WorkingDirectory</key>
    <string>/usr/local/opt/upscale-service</string>
    
    <key>RunAtLoad</key>
    <true/>
    
    <key>KeepAlive</key>
    <true/>
    
    <key>StandardErrorPath</key>
    <string>/usr/local/var/log/upscale-service.err</string>
    
    <key>StandardOutPath</key>
    <string>/usr/local/var/log/upscale-service.out</string>
</dict>
</plist>
```

**Installation:**
```bash
sudo mkdir -p /usr/local/opt/upscale-service
sudo cp -r build/* /usr/local/opt/upscale-service/
sudo cp deployments/launchd/com.upscale.service.plist /Library/LaunchDaemons/
sudo launchctl load /Library/LaunchDaemons/com.upscale.service.plist
```

### 8.3 Windows Service (deployments/windows/install-service.ps1)

```powershell
# PowerShell script to install as Windows Service
# Requires NSSM (Non-Sucking Service Manager)

$ServiceName = "UpscaleService"
$BinaryPath = "C:\upscale-service\upscale-server.exe"
$ConfigPath = "C:\upscale-service\config\config.yaml"

# Download NSSM if not present
if (!(Test-Path "nssm.exe")) {
    Write-Host "Downloading NSSM..."
    Invoke-WebRequest -Uri "https://nssm.cc/release/nssm-2.24.zip" -OutFile "nssm.zip"
    Expand-Archive "nssm.zip"
    Copy-Item "nssm\win64\nssm.exe" .
}

# Install service
Write-Host "Installing service..."
.\nssm.exe install $ServiceName $BinaryPath "-config" $ConfigPath

# Configure service
.\nssm.exe set $ServiceName AppDirectory "C:\upscale-service"
.\nssm.exe set $ServiceName DisplayName "Image Upscale Service"
.\nssm.exe set $ServiceName Description "AI-powered image upscaling service"
.\nssm.exe set $ServiceName Start SERVICE_AUTO_START

# Start service
.\nssm.exe start $ServiceName

Write-Host "Service installed and started successfully"
```

---

## 9. Client Implementation

### 9.1 Go Client (cmd/client/main.go)

```go
package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "time"
)

type Client struct {
    serverURL  string
    httpClient *http.Client
}

func NewClient(serverURL string) *Client {
    return &Client{
        serverURL: serverURL,
        httpClient: &http.Client{
            Timeout: 15 * time.Minute,
        },
    }
}

func (c *Client) Upscale(inputPath, outputPath string, scale int, model string) error {
    file, err := os.Open(inputPath)
    if err != nil {
        return fmt.Errorf("failed to open input: %w", err)
    }
    defer file.Close()
    
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    
    part, err := writer.CreateFormFile("image", filepath.Base(inputPath))
    if err != nil {
        return err
    }
    io.Copy(part, file)
    
    writer.WriteField("scale", fmt.Sprintf("%d", scale))
    if model != "" {
        writer.WriteField("model_name", model)
    }
    writer.Close()
    
    resp, err := c.httpClient.Post(
        c.serverURL+"/api/v1/upscale",
        writer.FormDataContentType(),
        body,
    )
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("failed to parse response: %w", err)
    }
    
    if !result["success"].(bool) {
        return fmt.Errorf("upscale failed: %s", result["error"])
    }
    
    fmt.Printf("Duration: %.2fs\n", result["duration_seconds"])
    fmt.Printf("Input: %v\n", result["input_size"])
    fmt.Printf("Output: %v\n", result["output_size"])
    
    downloadURL := c.serverURL + result["download_url"].(string)
    return c.download(downloadURL, outputPath)
}

func (c *Client) download(url, outputPath string) error {
    resp, err := c.httpClient.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    out, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer out.Close()
    
    _, err = io.Copy(out, resp.Body)
    return err
}

func (c *Client) ListModels() ([]interface{}, error) {
    resp, err := c.httpClient.Get(c.serverURL + "/api/v1/models")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result["models"].([]interface{}), nil
}

func main() {
    serverURL := flag.String("server", "http://localhost:8080", "Server URL")
    inputFile := flag.String("input", "", "Input image file")
    outputFile := flag.String("output", "upscaled.png", "Output image file")
    scale := flag.Int("scale", 4, "Scale factor")
    model := flag.String("model", "realesrgan-x4plus", "Model name")
    listModels := flag.Bool("list-models", false, "List available models")
    
    flag.Parse()
    
    client := NewClient(*serverURL)
    
    if *listModels {
        models, err := client.ListModels()
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            os.Exit(1)
        }
        
        fmt.Println("Available models:")
        for _, m := range models {
            model := m.(map[string]interface{})
            fmt.Printf("  - %s: %s\n", model["name"], model["description"])
        }
        return
    }
    
    if *inputFile == "" {
        fmt.Println("Error: -input required")
        flag.PrintDefaults()
        os.Exit(1)
    }
    
    fmt.Printf("Upscaling %s...\n", *inputFile)
    
    if err := client.Upscale(*inputFile, *outputFile, *scale, *model); err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Printf("Saved to %s\n", *outputFile)
}
```

### 9.2 Python Client (optional)

```python
#!/usr/bin/env python3
import argparse
import requests
from pathlib import Path

class UpscaleClient:
    def __init__(self, server_url):
        self.server_url = server_url
        
    def upscale(self, input_path, output_path, scale=4, model="realesrgan-x4plus"):
        with open(input_path, 'rb') as f:
            files = {'image': f}
            data = {'scale': scale, 'model_name': model}
            
            response = requests.post(
                f"{self.server_url}/api/v1/upscale",
                files=files,
                data=data,
                timeout=900
            )
            
        result = response.json()
        
        if not result['success']:
            raise Exception(result['error'])
            
        print(f"Duration: {result['duration_seconds']:.2f}s")
        print(f"Input: {result['input_size']}")
        print(f"Output: {result['output_size']}")
        
        download_url = self.server_url + result['download_url']
        
        response = requests.get(download_url)
        with open(output_path, 'wb') as f:
            f.write(response.content)

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--server', default='http://localhost:8080')
    parser.add_argument('--input', required=True)
    parser.add_argument('--output', default='upscaled.png')
    parser.add_argument('--scale', type=int, default=4)
    parser.add_argument('--model', default='realesrgan-x4plus')
    
    args = parser.parse_args()
    
    client = UpscaleClient(args.server)
    client.upscale(args.input, args.output, args.scale, args.model)
    
    print(f"Saved to {args.output}")
```

---

## 10. Testing

### 10.1 Unit Tests

```go
// internal/upscaler/upscaler_test.go
package upscaler

import (
    "context"
    "os"
    "testing"
)

func TestValidation(t *testing.T) {
    s := NewService(Config{
        BinaryPath: "/usr/bin/test",
        ModelsPath: "/tmp/models",
    })
    
    tests := []struct {
        name    string
        req     Request
        wantErr bool
    }{
        {
            name: "valid request",
            req: Request{
                InputPath: "/tmp/test.jpg",
                Scale:     4,
                ModelName: "realesrgan-x4plus",
            },
            wantErr: false,
        },
        {
            name: "invalid scale",
            req: Request{
                InputPath: "/tmp/test.jpg",
                Scale:     5,
                ModelName: "realesrgan-x4plus",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := s.validate(tt.req)
            if (err != nil) != tt.wantErr {
                t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 10.2 Integration Tests

```go
// test/integration/api_test.go
package integration

import (
    "bytes"
    "encoding/json"
    "mime/multipart"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestUpscaleAPI(t *testing.T) {
    // Setup test server
    router := setupTestRouter()
    
    // Create test request
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    
    // Add test image
    // ... (add test image data)
    
    writer.WriteField("scale", "4")
    writer.Close()
    
    req := httptest.NewRequest("POST", "/api/v1/upscale", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
    
    var result map[string]interface{}
    json.NewDecoder(w.Body).Decode(&result)
    
    if !result["success"].(bool) {
        t.Errorf("Expected success=true, got %v", result)
    }
}
```

---

## 11. Deployment-Workflow

### Quick Start (Docker)

```bash
# 1. Clone repository
git clone https://github.com/yourusername/upscale-service.git
cd upscale-service

# 2. Download models
./scripts/download-models.sh

# 3. Build Docker image
docker-compose build

# 4. Start service
docker-compose up -d

# 5. Test
curl http://localhost:8080/api/v1/health
```

### Production Deployment (Docker)

```bash
# 1. Build production image
docker build -t upscale-service:1.0.0 -f deployments/docker/Dockerfile .

# 2. Tag for registry
docker tag upscale-service:1.0.0 registry.example.com/upscale-service:1.0.0

# 3. Push to registry
docker push registry.example.com/upscale-service:1.0.0

# 4. Deploy to production
docker pull registry.example.com/upscale-service:1.0.0
docker run -d \
  --name upscale-service \
  -p 8080:8080 \
  -v /data/models:/app/models:ro \
  -v /data/uploads:/app/data/uploads \
  -v /data/outputs:/app/data/outputs \
  --restart unless-stopped \
  registry.example.com/upscale-service:1.0.0
```

### Native Installation (Linux)

```bash
# 1. Build for Linux
./scripts/build.sh

# 2. Install
sudo mkdir -p /opt/upscale-service
sudo cp -r build/* /opt/upscale-service/
sudo ./scripts/download-models.sh

# 3. Install systemd service
sudo cp deployments/systemd/upscale-service.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable upscale-service
sudo systemctl start upscale-service

# 4. Check status
sudo systemctl status upscale-service
```

---

## 12. Monitoring & Logging

### Logging

- **Format**: JSON (structured logging)
- **Levels**: debug, info, warn, error
- **Output**: stdout (Docker), file (systemd)

### Metrics (optional - future)

```go
// Prometheus metrics
- upscale_requests_total
- upscale_duration_seconds
- upscale_errors_total
- upscale_queue_size
- upscale_active_jobs
```

### Health Checks

```bash
# Docker
docker exec upscale-service curl http://localhost:8080/api/v1/health

# Direct
curl http://localhost:8080/api/v1/health
```

---

## 13. Makefile

```makefile
.PHONY: help build test clean docker-build docker-run install models

help:
	@echo "Available targets:"
	@echo "  build         - Build Go binary"
	@echo "  test          - Run tests"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with Docker Compose"
	@echo "  models        - Download AI models"
	@echo "  install       - Install as systemd service (Linux)"
	@echo "  clean         - Clean build artifacts"

build:
	@./scripts/build.sh

test:
	@go test -v ./...

docker-build:
	@./scripts/build-docker.sh

docker-run:
	@docker-compose up -d

models:
	@./scripts/download-models.sh

install: build models
	@sudo mkdir -p /opt/upscale-service
	@sudo cp -r build/* /opt/upscale-service/
	@sudo cp deployments/systemd/upscale-service.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@sudo systemctl enable upscale-service
	@sudo systemctl start upscale-service

clean:
	@rm -rf build/ dist/ data/uploads/* data/outputs/*
```

---

## 14. Timeline & Milestones

### Phase 1: Core Development (Week 1-2)
- [ ] Setup project structure
- [ ] Implement config loading
- [ ] Implement upscaler service
- [ ] Implement storage manager
- [ ] Implement API handlers
- [ ] Basic unit tests

### Phase 2: Docker & Deployment (Week 3)
- [ ] Create Dockerfile
- [ ] Create docker-compose.yml
- [ ] Build scripts
- [ ] Model download automation
- [ ] Integration tests

### Phase 3: Cross-Platform Services (Week 4)
- [ ] Linux systemd service
- [ ] macOS launchd service
- [ ] Windows service installer
- [ ] Documentation

### Phase 4: Polish & Release (Week 5)
- [ ] Client implementations
- [ ] API documentation
- [ ] Deployment guides
- [ ] CI/CD pipeline
- [ ] Release v1.0.0

---

## 15. Next Steps

1. **Initialize Go project**: `go mod init github.com/yourusername/upscale-service`
2. **Create directory structure**: Follow struktur above
3. **Download models**: Run `scripts/download-models.sh`
4. **Build ncnn**: Compile Real-ESRGAN-ncnn-vulkan
5. **Implement core**: Start with config → upscaler → API
6. **Test locally**: Run server, test with curl
7. **Dockerize**: Build Docker image
8. **Deploy**: Choose deployment method (Docker recommended)

---

## Zusammenfassung

Dieses Projekt bietet:

✅ **Docker-First**: Einfaches Deployment mit Docker/Compose
✅ **Cross-Platform**: Linux, macOS, Windows Support
✅ **Production-Ready**: Logging, Health-Checks, Graceful Shutdown
✅ **Scalable**: Job-Queue, Worker-Pools, Concurrency-Limits
✅ **Well-Structured**: Clean Architecture, Testable Code
✅ **Easy Integration**: REST API, Client-Libraries

**Hauptfokus**: Docker-Deployment, aber flexibel für native Services.

