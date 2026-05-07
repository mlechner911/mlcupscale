// Copyright (c) 2026 Michael Lechner
// MIT License

package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"upscale-service/internal/upscaler"
)

//go:embed assets/*
var assets embed.FS

func main() {
	input := flag.String("i", "", "Input image path")
	output := flag.String("o", "upscaled.png", "Output image path")
	scale := flag.Int("s", 4, "Upscale factor (2, 3, 4)")
	model := flag.String("n", "realesrgan-x4plus", "Model name")
	tileSize := flag.Int("t", 512, "Tile size")
	gpuID := flag.Int("g", 0, "GPU ID (-1 for CPU)")
	listModels := flag.Bool("list", false, "List embedded models")
	help := flag.Bool("h", false, "Show help")

	flag.Parse()

	if *help || (*input == "" && !*listModels) {
		flag.Usage()
		return
	}

	// 1. Setup local environment
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".mlcupscale-cli")
	binDir := filepath.Join(baseDir, "bin")
	modelDir := filepath.Join(baseDir, "models")

	if *listModels {
		fmt.Println("Embedded models:")
		entries, _ := assets.ReadDir("assets/models")
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".param") {
				fmt.Printf("  - %s\n", strings.TrimSuffix(e.Name(), ".param"))
			}
		}
		return
	}

	fmt.Printf("Preparing environment in %s...\n", baseDir)
	if err := extractAssets(binDir, modelDir); err != nil {
		fmt.Printf("Error preparing environment: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize Upscaler
	binaryName := "realesrgan-ncnn-vulkan"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binPath := filepath.Join(binDir, binaryName)

	cfg := upscaler.Config{
		BinaryPath:   binPath,
		ModelsPath:   modelDir,
		DefaultModel: "realesrgan-x4plus",
		DefaultScale: 4,
		Threads:      "1:2:2",
		EnableGPU:    *gpuID >= 0,
		GPUID:        *gpuID,
	}

	svc := upscaler.NewService(cfg)

	// 3. Run Upscale
	req := upscaler.Request{
		InputPath:  *input,
		OutputPath: *output,
		Scale:      *scale,
		ModelName:  *model,
		TileSize:   *tileSize,
	}

	fmt.Printf("Upscaling %s (%dx) using %s...\n", *input, *scale, *model)

	onProgress := func(p int) {
		fmt.Printf("\rProgress: %d%%   ", p)
	}

	ctx := context.Background()
	result, err := svc.Upscale(ctx, req, onProgress)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nDone! Saved to %s (took %.2fs)\n", result.OutputPath, result.Duration.Seconds())
}

func extractAssets(binDir, modelDir string) error {
	os.MkdirAll(binDir, 0755)
	os.MkdirAll(modelDir, 0755)

	// Extract Binaries
	binSuffix := ""
	if runtime.GOOS == "windows" {
		binSuffix = ".exe"
	} else if runtime.GOOS == "darwin" {
		binSuffix = "-macos"
	}
	
	srcBin := "assets/bin/realesrgan-ncnn-vulkan" + binSuffix
	dstBin := filepath.Join(binDir, "realesrgan-ncnn-vulkan")
	if runtime.GOOS == "windows" {
		dstBin += ".exe"
	}

	if err := copyEmbed(srcBin, dstBin, 0755); err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}

	// Extract Models
	entries, err := assets.ReadDir("assets/models")
	if err != nil {
		return err
	}

	for _, e := range entries {
		// Use simple string joining with / for embedded FS
		src := "assets/models/" + e.Name()
		dst := filepath.Join(modelDir, e.Name())
		if err := copyEmbed(src, dst, 0644); err != nil {
			return fmt.Errorf("failed to extract model %s: %w", e.Name(), err)
		}
	}

	return nil
}

func copyEmbed(src, dst string, mode os.FileMode) error {
	// Check if already exists and size matches to avoid re-extracting
	if _, err := os.Stat(dst); err == nil {
		// For simplicity, we just overwrite if it's a small binary or if we want to be sure
		// but a real tool might check version/hash.
	}

	data, err := assets.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, mode)
}

