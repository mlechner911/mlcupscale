// Copyright (c) 2026 Michael Lechner
// MIT License

package api

import (
    "fmt"
    "net/http"
    "path/filepath"
    "time"
    
    "github.com/gin-gonic/gin"
    
    "upscale-service/internal/storage"
    "upscale-service/internal/upscaler"
)

const ServiceVersion = "1.0.0"

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
    
    // Generate output path
    preID := fmt.Sprintf("%d", time.Now().UnixNano())
    outputPath := h.storage.GetOutputPath(preID, fileHeader.Filename)

    jobID, err := h.upscaler.SubmitJob(upscaler.Request{
        InputPath:  inputPath,
        OutputPath: outputPath,
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
    
    // Wait for completion (sync mode for now)
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
                errMsg := "unknown error"
                if job.Error != nil {
                    errMsg = job.Error.Error()
                }
                c.JSON(http.StatusInternalServerError, UpscaleResponse{
                    Success: false,
                    Error:   errMsg,
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
    
    // Ensure the file exists
    if _, err := http.Dir(filepath.Dir(job.Result.OutputPath)).Open(filepath.Base(job.Result.OutputPath)); err != nil {
         c.JSON(http.StatusNotFound, gin.H{"error": "output file missing"})
         return
    }

    c.File(job.Result.OutputPath)
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

    // Filter by scale if requested
    scaleStr := c.Query("scale")
    if scaleStr != "" {
        var scale int
        if _, err := fmt.Sscanf(scaleStr, "%d", &scale); err == nil {
             filtered := make([]upscaler.ModelInfo, 0)
             for _, m := range models {
                 for _, s := range m.SupportedScales {
                     if s == scale {
                         filtered = append(filtered, m)
                         break
                     }
                 }
             }
             models = filtered
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "version": ServiceVersion,
        "models":  models,
    })
}

func (h *Handler) HandleHealth(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":  "ok",
        "version": ServiceVersion,
        "time":    time.Now().Unix(),
    })
}