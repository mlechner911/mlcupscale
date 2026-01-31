// Copyright (c) 2026 Michael Lechner
// MIT License

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"upscale-service/internal/api"
	"upscale-service/internal/config"
	"upscale-service/internal/storage"
	"upscale-service/internal/upscaler"
	"upscale-service/internal/version"
)

var (
    configPath = flag.String("config", "config/config.yaml", "Path to config file")
)

// main is the entry point for the upscale server.
// It loads configuration, initializes services, sets up the API router, and starts the HTTP server.
func main() {
    flag.Parse()

    log.Printf("Starting Upscale Service v%s", version.Version)

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
        UploadDir:       cfg.Storage.UploadDir,
        OutputDir:       cfg.Storage.OutputDir,
        MaxFileSizeMB:   cfg.Storage.MaxFileSizeMB,
        CleanupTTL:      15 * time.Minute, // Hardcoded to 15 mins per requirement
        RetentionPolicy: cfg.Storage.RetentionPolicy,
    })
    if err != nil {
        log.Fatalf("Failed to initialize storage: %v", err)
    }

    // Start cleanup routine
    go func() {
        // Run cleanup every minute
        ticker := time.NewTicker(1 * time.Minute)
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
        router.Use(api.CORSMiddleware(cfg.Features.AllowedOrigins))
    }

    // API Group with optional Auth
    apiGroup := router.Group(cfg.Server.APIPrefix)

    if cfg.Server.AuthToken != "" {
        log.Println("Authentication enabled")
        apiGroup.Use(api.AuthMiddleware(cfg.Server.AuthToken))
    }

    {
        apiGroup.POST("/upscale", handler.HandleUpscale)
        apiGroup.GET("/download/:job_id", handler.HandleDownload)
        apiGroup.GET("/status/:job_id", handler.HandleStatus)
        apiGroup.POST("/cancel/:job_id", handler.HandleCancel)
        apiGroup.GET("/models", handler.HandleModels)
        apiGroup.GET("/health", handler.HandleHealth)
    }

    // Swagger UI
    if cfg.Features.EnableSwagger {
        log.Printf("Swagger UI enabled at %s/docs", cfg.Server.APIPrefix)
        docsDir := filepath.Join(getBaseDir(), "docs")

        // Serve the OpenAPI spec file
        apiGroup.StaticFile("/openapi.yaml", filepath.Join(docsDir, "openapi.yaml"))

        // Serve the Swagger UI HTML
        apiGroup.StaticFile("/docs", filepath.Join(docsDir, "swagger.html"))
    }

    // Start server
    addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    log.Printf("Server listening on %s%s", addr, cfg.Server.APIPrefix)

    if err := router.Run(addr); err != nil {
        log.Fatal(err)
    }
}

// getBaseDir returns the directory where the executable is located.
// It is used to resolve relative paths for configuration and assets.
func getBaseDir() string {
    exe, err := os.Executable()
    if err != nil {
        return "."
    }
    // If running with "go run", the executable is in a temp dir,
    // so we might want to fallback to "." or handle it differently.
    // For now, assume if we are in a temp dir (typical for go run), we use CWD.
    // However, the plan provided this implementation.
    // A robust way often checks if "config" exists in the dir of the executable.

    dir := filepath.Dir(exe)
    if _, err := os.Stat(filepath.Join(dir, "config")); os.IsNotExist(err) {
        // Fallback to CWD
        wd, _ := os.Getwd()
        return wd
    }
    return dir
}
