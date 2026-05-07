// Copyright (c) 2026 Michael Lechner
// MIT License

package upscaler

import (
	"bufio"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"upscale-service/internal/osutil"

	_ "golang.org/x/image/webp"
)

// Config holds the configuration settings for the upscaler service.
type Config struct {
	BinaryPath     string
	OnnxBinaryPath string
	NcnnBinaryPath string
	ModelsPath     string
	DefaultModel   string
	DefaultScale   int
	Threads        string
    EnableGPU    bool
    GPUID        int
}

// Request represents a single image upscaling task request.
type Request struct {
    InputPath  string
    OutputPath string
    Scale      int
    ModelName  string
    TileSize   int
    Format     string
}

// Result contains the output information of a completed upscaling task.
type Result struct {
    OutputPath    string
    Duration      time.Duration
    InputSize     ImageSize
    OutputSize    ImageSize
    FileSizeBytes int64
}

// ImageSize represents the dimensions of an image.
type ImageSize struct {
    Width  int `json:"width"`
    Height int `json:"height"`
}

// Job represents an asynchronous upscaling job managed by the service.
type Job struct {
    ID         string
    Request    Request
    Status     string
    Progress   int
    StartTime  time.Time
    Result     *Result
    Error      error
    cancelFunc context.CancelFunc
}

// Service manages the upscaling queue and execution.
type Service struct {
    config   Config
    jobs     map[string]*Job
    jobsMu   sync.Mutex
    jobQueue chan *Job
}

// NewService creates a new upscaler service instance.
func NewService(cfg Config) *Service {
    return &Service{
        config:   cfg,
        jobs:     make(map[string]*Job),
        jobQueue: make(chan *Job, 100),
    }
}

// StartWorkers starts the specified number of worker goroutines.
func (s *Service) StartWorkers(count int) {
    for i := 0; i < count; i++ {
        go s.worker()
    }
}

// worker processes jobs from the queue.
func (s *Service) worker() {
    for job := range s.jobQueue {
        s.processJob(job)
    }
}

// SubmitJob adds a new upscaling request to the processing queue.
func (s *Service) SubmitJob(req Request) (string, error) {
    s.jobsMu.Lock()

    id := generateJobID()
    job := &Job{
        ID:        id,
        Request:   req,
        Status:    "queued",
        Progress:  0,
        StartTime: time.Now(),
    }

    s.jobs[id] = job
    s.jobsMu.Unlock()

    // Send to worker queue
    // Use a goroutine to avoid blocking the API response if the queue is full
    go func() {
        s.jobQueue <- job
    }()

    return id, nil
}

// GetJob retrieves the status and details of a specific job.
func (s *Service) GetJob(jobID string) (*Job, bool) {
    s.jobsMu.Lock()
    defer s.jobsMu.Unlock()

    job, ok := s.jobs[jobID]
    return job, ok
}

// processJob executes the upscaling logic for a given job and updates its status.
func (s *Service) processJob(job *Job) {
    s.jobsMu.Lock()
    // Check if already cancelled while in queue
    if job.Status == "cancelled" {
        s.jobsMu.Unlock()
        return
    }

    job.Status = "processing"
    job.Progress = 1 // Set to 1% immediately

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    job.cancelFunc = cancel
    s.jobsMu.Unlock()

    defer cancel()

    // Progress callback
    onProgress := func(p int) {
        s.jobsMu.Lock()
        if p > job.Progress {
            job.Progress = p
        }
        s.jobsMu.Unlock()
    }

    // Prepare temp working directory and copy input/output so current files
    // are visible under ~/.mlcupscale/tmp while processing.
    req := job.Request
    origInput := req.InputPath
    origOutput := req.OutputPath

    homeDir, errHome := os.UserHomeDir()
    if errHome != nil {
        homeDir = os.TempDir()
    }

    tmpDir := filepath.Join(homeDir, ".mlcupscale", "tmp")
    if err := os.MkdirAll(tmpDir, 0755); err != nil {
        s.jobsMu.Lock()
        job.Status = "failed"
        job.Error = fmt.Errorf("failed to create temp dir: %w", err)
        s.jobsMu.Unlock()
        return
    }

    tmpInput := filepath.Join(tmpDir, job.ID+"_"+filepath.Base(origInput))
    if err := copyFile(origInput, tmpInput); err != nil {
        s.jobsMu.Lock()
        job.Status = "failed"
        job.Error = fmt.Errorf("failed to copy input to temp: %w", err)
        s.jobsMu.Unlock()
        return
    }

    tmpOutput := filepath.Join(tmpDir, job.ID+"_out_"+filepath.Base(origOutput))
    req.InputPath = tmpInput
    req.OutputPath = tmpOutput

    result, err := s.Upscale(ctx, req, onProgress)

    // If upscale succeeded, move tmp output back to original output location
    if err == nil && result != nil {
        // Ensure destination dir exists
        _ = os.MkdirAll(filepath.Dir(origOutput), 0755)

        // Try rename, fall back to copy
        if mvErr := os.Rename(tmpOutput, origOutput); mvErr != nil {
            if cpErr := copyFile(tmpOutput, origOutput); cpErr != nil {
                s.jobsMu.Lock()
                job.Status = "failed"
                job.Error = fmt.Errorf("failed to move output to final location: rename=%v copy=%v", mvErr, cpErr)
                s.jobsMu.Unlock()
                return
            }
            _ = os.Remove(tmpOutput)
        }

        result.OutputPath = origOutput
    }

    // Remove temp input file
    _ = os.Remove(tmpInput)

    s.jobsMu.Lock()
    defer s.jobsMu.Unlock()

    job.cancelFunc = nil // Cleanup

    if job.Status == "cancelled" {
        // Already marked as cancelled by CancelJob
        return
    }

    if err != nil {
        // Check if error was due to context cancellation
        if ctx.Err() == context.Canceled {
             job.Status = "cancelled"
        } else {
             job.Status = "failed"
             job.Error = err
        }
    } else {
        job.Status = "completed"
        job.Progress = 100
        job.Result = result
    }
}

// CancelJob attempts to cancel a running or queued job.
func (s *Service) CancelJob(jobID string) error {
    s.jobsMu.Lock()
    defer s.jobsMu.Unlock()

    job, ok := s.jobs[jobID]
    if !ok {
        return fmt.Errorf("job not found")
    }

    if job.Status == "completed" || job.Status == "failed" {
        return fmt.Errorf("job already finished")
    }

    if job.Status == "cancelled" {
        return nil
    }

    // If running, cancel the context
    if job.cancelFunc != nil {
        job.cancelFunc()
    }

    job.Status = "cancelled"
    return nil
}

// Upscale performs the actual image upscaling using the external binary.
func (s *Service) Upscale(ctx context.Context, req Request, onProgress func(int)) (*Result, error) {
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

    // Determine Engine and Binary
    var binaryPath string
    var isOnnx bool

    // Check for ONNX model first
    onnxPath := filepath.Join(s.config.ModelsPath, req.ModelName+".onnx")
    if _, err := os.Stat(onnxPath); err == nil {
        isOnnx = true
        // Prefer specific ONNX binary, fallback to general binary path
        if s.config.OnnxBinaryPath != "" {
            binaryPath = s.config.OnnxBinaryPath
        } else {
            binaryPath = s.config.BinaryPath
        }
    } else {
        // Assume NCNN
        isOnnx = false
        if s.config.NcnnBinaryPath != "" {
            binaryPath = s.config.NcnnBinaryPath
        } else {
            binaryPath = s.config.BinaryPath
        }
    }

    if binaryPath == "" {
        return nil, fmt.Errorf("appropriate upscaler binary not configured for model type (onnx=%v)", isOnnx)
    }

    // Build command
    args := s.buildArgs(req, isOnnx)

    cmd := osutil.PrepareCommand(exec.CommandContext(ctx, binaryPath, args...))

    // Capture stderr for progress
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
    }

    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("failed to start command: %w", err)
    }

    // Parse progress in a goroutine
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()

        scanner := bufio.NewScanner(stderr)
        scanner.Split(bufio.ScanLines)

        for scanner.Scan() {
            line := scanner.Text()
            // Debug: Log binary output
            fmt.Println("Upscaler Output:", line)

            // Format is typically "23.45%"
            if strings.Contains(line, "%") {
                line = strings.TrimSpace(line)
                line = strings.TrimSuffix(line, "%")

                var percent float64
                if _, err := fmt.Sscanf(line, "%f", &percent); err == nil {
                    if onProgress != nil {
                        onProgress(int(percent))
                    }
                }
            }
        }
    }()

    // Wait for completion
    if err := cmd.Wait(); err != nil {
        return nil, fmt.Errorf("upscale failed: %w", err)
    }

    wg.Wait()

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

// validate checks if the request parameters and required files are valid.
func (s *Service) validate(req Request) error {
	// We no longer strictly check s.config.BinaryPath here because it might vary per model
	// But we should check that at least one binary is configured.
	if s.config.BinaryPath == "" && s.config.OnnxBinaryPath == "" && s.config.NcnnBinaryPath == "" {
		return fmt.Errorf("no upscaler binary configured")
	}

	if _, err := os.Stat(req.InputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", req.InputPath)
	}

	if req.Scale < 2 || req.Scale > 4 {
		return fmt.Errorf("invalid scale: %d (must be 2, 3, or 4)", req.Scale)
	}

	modelPath := filepath.Join(s.config.ModelsPath, req.ModelName)
	// Check if ONNX
	hasOnnx := false
	if _, err := os.Stat(modelPath + ".onnx"); err == nil {
		hasOnnx = true
	}

	// Check if standard NCNN
	hasNcnn := false
	if _, err := os.Stat(modelPath); err == nil {
		hasNcnn = true
	} else if _, err := os.Stat(modelPath + ".param"); err == nil {
		hasNcnn = true
	}

	if !hasOnnx && !hasNcnn {
		return fmt.Errorf("model not found: %s", req.ModelName)
	}

	return nil
}

// buildArgs constructs the command-line arguments for the upscaler binary.
func (s *Service) buildArgs(req Request, isOnnx bool) []string {
	args := []string{
		"-i", req.InputPath,
		"-o", req.OutputPath,
		"-s", fmt.Sprintf("%d", req.Scale),
		"-m", s.config.ModelsPath,
		"-n", req.ModelName,
		"-j", s.config.Threads,
	}

	if isOnnx {
		// ONNX implementation needs -t 512 for stability
		args = append(args, "-t", "512")
	} else {
		// NCNN implementation might not need it, or supports it differently
		// But for now, we only forced it for ONNX/CoreML stability.
		// If we want consistency, we can add it here too if the binary supports it.
		// The standard realesrgan-ncnn-vulkan DOES support -t.
		args = append(args, "-t", "512")
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

// getImageSize reads the image dimensions from a file.
func (s *Service) getImageSize(path string) (ImageSize, error) {
	file, err := os.Open(path)
	if err != nil {
		return ImageSize{}, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return ImageSize{}, fmt.Errorf("failed to decode image config: %w", err)
	}

	return ImageSize{Width: cfg.Width, Height: cfg.Height}, nil
}


// GetAvailableModels scans the models directory and returns a list of installed models.
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

        // Check extensions
        if strings.HasSuffix(name, ".onnx") {
            modelName = strings.TrimSuffix(name, ".onnx")
        } else if strings.HasSuffix(name, ".param") {
            modelName = strings.TrimSuffix(name, ".param")
        } else if strings.HasSuffix(name, ".bin") {
             continue // skip bin, we count the param
        } else if entry.IsDir() {
             // Directories are often models too
        } else {
             continue
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

// ModelInfo describes the capabilities of an AI model.
type ModelInfo struct {
    Name            string `json:"name"`
    Description     string `json:"description"`
    SupportedScales []int  `json:"supported_scales"`
}

// copyFile copies a file from src to dst preserving mode where possible.
func copyFile(src, dst string) error {
    in, err := os.Open(src)
    if err != nil {
        return err
    }
    defer in.Close()

    out, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer func() {
        _ = out.Close()
    }()

    if _, err := io.Copy(out, in); err != nil {
        return err
    }

    if fi, err := in.Stat(); err == nil {
        _ = out.Chmod(fi.Mode())
    }

    if err := out.Sync(); err != nil {
        return err
    }

    return out.Close()
}

// generateJobID creates a unique identifier for a job based on the current timestamp.
func generateJobID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}