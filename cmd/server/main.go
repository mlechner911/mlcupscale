// Copyright (c) 2026 Michael Lechner
// MIT License

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"io"

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
    // Robust search: if not found, check in EXE_DIR/config.yaml or EXE_DIR/config/config.yaml
    path := *configPath
    if _, err := os.Stat(path); os.IsNotExist(err) {
        log.Printf("Config not found at %s, searching in executable directory...", path)
        base := getBaseDir()
        altPaths := []string{
            filepath.Join(base, "config.yaml"),
            filepath.Join(base, "config", "config.yaml"),
        }
        for _, alt := range altPaths {
            if _, err := os.Stat(alt); err == nil {
                path = alt
                log.Printf("Found config at %s", path)
                break
            }
        }
    }

    cfg, err := config.Load(path)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    setupLogging(cfg)

    // Make paths absolute if relative
    if !filepath.IsAbs(cfg.Upscaler.BinaryPath) {
        cfg.Upscaler.BinaryPath = filepath.Join(getBaseDir(), cfg.Upscaler.BinaryPath)
    }
    if !filepath.IsAbs(cfg.Upscaler.ModelsPath) {
        cfg.Upscaler.ModelsPath = filepath.Join(getBaseDir(), cfg.Upscaler.ModelsPath)
    }

    // Make paths absolute if relative
    baseDir := getBaseDir()
    dataDir := baseDir

    // On Windows, if we are in Program Files, we should use a user-writable directory for data
    if isSystemDir(baseDir) {
        userDir, err := os.UserConfigDir()
        if err == nil {
            dataDir = filepath.Join(userDir, "MLCUpscale")
            log.Printf("Running from system directory, using user data directory: %s", dataDir)
        }
    }

    if !filepath.IsAbs(cfg.Storage.UploadDir) {
        cfg.Storage.UploadDir = filepath.Join(dataDir, cfg.Storage.UploadDir)
    }
    if !filepath.IsAbs(cfg.Storage.OutputDir) {
        cfg.Storage.OutputDir = filepath.Join(dataDir, cfg.Storage.OutputDir)
    }

    // Ensure directories exist and are writable
    for _, d := range []string{cfg.Storage.UploadDir, cfg.Storage.OutputDir} {
        if err := os.MkdirAll(d, 0755); err != nil {
            log.Fatalf("Failed to create directory %s: %v. Please ensure the application has write permissions.", d, err)
        }
        // Test writability
        testFile := filepath.Join(d, ".write_test")
        if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
            log.Fatalf("Directory %s is not writable: %v. Please run as administrator or configure a different storage path.", d, err)
        }
        _ = os.Remove(testFile)
    }


    // Initialize services
    upscalerService := upscaler.NewService(upscaler.Config{
        BinaryPath:     cfg.Upscaler.BinaryPath,
        OnnxBinaryPath: cfg.Upscaler.OnnxBinaryPath,
        NcnnBinaryPath: cfg.Upscaler.NcnnBinaryPath,
        ModelsPath:     cfg.Upscaler.ModelsPath,
        DefaultModel:   cfg.Upscaler.DefaultModel,
        DefaultScale:   cfg.Upscaler.DefaultScale,
        Threads:        cfg.Upscaler.Threads,
        EnableGPU:      cfg.Upscaler.EnableGPU,
        GPUID:          cfg.Upscaler.GPUID,
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

    // OpenAI Compatibility
    router.GET("/v1/models", handler.HandleOpenAIModels)

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

// isSystemDir checks if the path is in a restricted system directory (e.g. Program Files on Windows).
func isSystemDir(path string) bool {
    p := strings.ToLower(path)
    
    // On Windows, check for Program Files and Windows directory
    if runtime.GOOS == "windows" {
        progFiles := strings.ToLower(os.Getenv("ProgramFiles"))
        progFilesX86 := strings.ToLower(os.Getenv("ProgramFiles(x86)"))
        winDir := strings.ToLower(os.Getenv("SystemRoot"))
        
        if (progFiles != "" && strings.HasPrefix(p, progFiles)) ||
           (progFilesX86 != "" && strings.HasPrefix(p, progFilesX86)) ||
           (winDir != "" && strings.HasPrefix(p, winDir)) {
            return true
        }
    }
    
    // Fallback/Generic check
    return strings.Contains(p, "c:\\program files") || strings.Contains(p, "c:\\windows")
}

func getBaseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// setupLogging ensures logs are written to a file on Windows if not running interactively
func setupLogging(cfg *config.Config) {
	if runtime.GOOS == "windows" {
		logDir := filepath.Join(getBaseDir(), "logs")
		_ = os.MkdirAll(logDir, 0755)
		logFile := filepath.Join(logDir, "server.log")
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			log.SetOutput(io.MultiWriter(os.Stdout, f))
			log.Printf("Logging to %s", logFile)
		}
	}
}
