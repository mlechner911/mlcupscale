// Copyright (c) 2026 Michael Lechner
// MIT License

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
    s.jobsMu.Lock()
    job.Status = "processing"
    s.jobsMu.Unlock()
    
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
        // Log warning but continue? No, better to fail if we can't read it
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
        // Try with .param and .bin extensions if directory check fails
        // RealESRGAN models usually come as .param and .bin files with the same name
        if _, err := os.Stat(modelPath + ".param"); os.IsNotExist(err) {
             return fmt.Errorf("model not found: %s", req.ModelName)
        }
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
    // Requires ImageMagick
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
    seen := make(map[string]bool)

    for _, entry := range entries {
        // Filter for .param or .bin or directories
        name := entry.Name()
        modelName := name
        
        // If file, strip extension to get model name
        if !entry.IsDir() {
            ext := filepath.Ext(name)
            if ext == ".param" || ext == ".bin" {
                modelName = name[0 : len(name)-len(ext)]
            } else {
                continue
            }
        }
        
        if seen[modelName] {
            continue
        }
        seen[modelName] = true

        info := ModelInfo{
            Name:            modelName,
            SupportedScales: []int{2, 3, 4},
        }
        
        switch modelName {
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
