// Copyright (c) 2026 Michael Lechner
// MIT License

package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"upscale-service/internal/storage"
	"upscale-service/internal/upscaler"
	"upscale-service/internal/version"
)

// Handler manages the HTTP requests for the upscaling service.
// It coordinates between the Gin web framework, the upscaler service, and the storage manager.
type Handler struct {
    upscaler *upscaler.Service
    storage  *storage.Manager
}

// NewHandler creates a new instance of the Handler with the provided dependencies.
func NewHandler(upscalerService *upscaler.Service, storageManager *storage.Manager) *Handler {
    return &Handler{
        upscaler: upscalerService,
        storage:  storageManager,
    }
}

// UpscaleRequest represents the form data parameters for an upscale request.
type UpscaleRequest struct {
    // Scale factor for the image (2, 3, or 4).
    Scale     int    `form:"scale" json:"scale"`
    // ModelName is the name of the AI model to use.
    ModelName string `form:"model_name" json:"model_name"`
    // TileSize is the tile size for processing (0 for auto).
    TileSize  int    `form:"tile_size" json:"tile_size"`
    // Format is the desired output format (png, jpg, webp).
    Format    string `form:"format" json:"format"`
}

// UpscaleResponse represents the JSON response returned by the upscale endpoint.
type UpscaleResponse struct {
    // Success indicates whether the request was successful.
    Success       bool                  `json:"success"`
    // JobID is the unique identifier for the upscale job.
    JobID         string                `json:"job_id,omitempty"`
    // StatusURL is the URL to check job status.
    StatusURL     string                `json:"status_url,omitempty"`
    // DownloadURL is the relative URL to download the processed image.
    DownloadURL   string                `json:"download_url,omitempty"`
    // Duration is the time taken to process the image in seconds.
    Duration      float64               `json:"duration_seconds,omitempty"`
    // InputSize contains the dimensions of the input image.
    InputSize     *upscaler.ImageSize   `json:"input_size,omitempty"`
    // OutputSize contains the dimensions of the output image.
    OutputSize    *upscaler.ImageSize   `json:"output_size,omitempty"`
    // FileSizeBytes is the size of the output file in bytes.
    FileSizeBytes int64                 `json:"file_size_bytes,omitempty"`
    // Error contains the error message if the request failed.
    Error         string                `json:"error,omitempty"`
}

// HandleUpscale processes the image upload and submits an upscaling job.
// It expects a multipart form request with an 'image' file and optional parameters.
func (h *Handler) HandleUpscale(c *gin.Context) {
    var req UpscaleRequest
    if err := c.ShouldBind(&req); err != nil {
        c.JSON(http.StatusBadRequest, UpscaleResponse{
            Success: false,
            Error:   fmt.Sprintf("invalid request: %v", err),
        })
        return
    }

    // Apply defaults
    if req.Scale == 0 {
        req.Scale = 4
    }
    if req.ModelName == "" {
        req.ModelName = "realesrgan-x4plus"
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

    // Async response
    c.JSON(http.StatusAccepted, UpscaleResponse{
        Success:   true,
        JobID:     jobID,
        StatusURL: "/api/v1/status/" + jobID,
    })
}

// HandleDownload serves the upscaled image file for a given job ID.
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

    // Serve the file
    // Enable more efficient file serving if possible
    // Note: Gin's c.File() handles Content-Type sniffing and efficient sending
    // but for large files over default connections, we might want to ensure specific headers

    // Explicitly disabling Gzip for this large binary response to avoid double-compression overhead
    c.Header("Content-Description", "File Transfer")
    c.Header("Content-Transfer-Encoding", "binary")
    c.Header("Content-Disposition", "attachment; filename="+filepath.Base(job.Result.OutputPath))
    c.Header("Content-Type", "application/octet-stream")

    // Manual streaming to ensure no buffering
    f, err := os.Open(job.Result.OutputPath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
        return
    }
    defer f.Close()

    fi, err := f.Stat()
    if err == nil {
        c.Header("Content-Length", fmt.Sprintf("%d", fi.Size()))
    }

    _, _ = io.Copy(c.Writer, f)

    // Delete after download if policy dictates
    if h.storage.ShouldDeleteAfterDownload() {
        // We run this in a goroutine to not block the response,
        // but we need to wait a tiny bit to ensure the file server has opened the file handle
        go func() {
            time.Sleep(1 * time.Second) // Small buffer
            if err := h.storage.DeleteFile(job.Result.OutputPath); err != nil {
                fmt.Printf("Failed to delete file after download: %v\n", err)
            }
        }()
    }
}

// HandleStatus returns the current status and progress of a specific job.
func (h *Handler) HandleStatus(c *gin.Context) {
    jobID := c.Param("job_id")

    job, ok := h.upscaler.GetJob(jobID)
    if !ok {
        c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
        return
    }

    response := gin.H{
        "job_id":   job.ID,
        "status":   job.Status,
        "progress": job.Progress,
    }

    if job.Status == "completed" && job.Result != nil {
        response["download_url"] = "/api/v1/download/" + job.ID
        response["duration_seconds"] = job.Result.Duration.Seconds()
        response["input_size"] = job.Result.InputSize
        response["output_size"] = job.Result.OutputSize
        response["file_size_bytes"] = job.Result.FileSizeBytes
    } else if job.Status == "failed" {
        errMsg := "unknown error"
        if job.Error != nil {
            errMsg = job.Error.Error()
        }
        response["error"] = errMsg
    }

    c.JSON(http.StatusOK, response)
}

// HandleCancel cancels a running or queued job.
func (h *Handler) HandleCancel(c *gin.Context) {
    jobID := c.Param("job_id")

    if err := h.upscaler.CancelJob(jobID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "job cancelled",
    })
}

// HandleModels returns a list of available AI models and their capabilities.
// It supports filtering by 'scale' query parameter.
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
        "version": version.Version,
        "models":  models,
    })
}

// HandleHealth provides a health check endpoint returning status, version, and server time.
func (h *Handler) HandleHealth(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":  "ok",
        "version": version.Version,
        "time":    time.Now().Unix(),
    })
}
