// Copyright (c) 2026 Michael Lechner
// MIT License

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
        if msg, ok := result["error"].(string); ok {
             return fmt.Errorf("upscale failed: %s", msg)
        }
        return fmt.Errorf("upscale failed: unknown error")
    }
    
    if duration, ok := result["duration_seconds"].(float64); ok {
        fmt.Printf("Duration: %.2fs\n", duration)
    }
    if input, ok := result["input_size"]; ok {
         fmt.Printf("Input: %v\n", input)
    }
    if output, ok := result["output_size"]; ok {
         fmt.Printf("Output: %v\n", output)
    }
    
    downloadPath, ok := result["download_url"].(string)
    if !ok {
        return fmt.Errorf("response missing download_url")
    }
    
    downloadURL := c.serverURL + downloadPath
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
    
    models, ok := result["models"].([]interface{})
    if !ok {
         return nil, fmt.Errorf("invalid models response format")
    }
    return models, nil
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
            modelMap, ok := m.(map[string]interface{})
            if ok {
                fmt.Printf("  - %s: %s\n", modelMap["name"], modelMap["description"])
            }
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
