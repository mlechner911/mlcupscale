# 🛠 Tech Stack & Constraints: MLC Upscale Service

## Kern-Versionen
- **Sprache:** Go 1.24+ (Nutzt moderne Features und Performance-Optimierungen).
- **Runtime:** Docker (Linux/Debian-based), Native macOS (Darwin/arm64).
- **AI-Engine:** Real-ESRGAN ncnn-vulkan (Hardware-beschleunigt via Vulkan).

## Bibliotheken (Erlaubt/Fixiert)
- **Web Framework:** `github.com/gin-gonic/gin` (Standard für MLC Go-Backends).
- **Config:** `gopkg.in/yaml.v3` für YAML-basiertes Configuration Management.
- **Image Processing:** `golang.org/x/image` für grundlegende Bildmanipulationen vor/nach dem Upscaling.

## Einschränkungen (Constraints)
- **Kein schweres ORM:** Job-Tracking erfolgt über ein effizientes In-Memory Management mit Persistenz auf dem Dateisystem (JSON-Meta-Files).
- **Cross-Platform Pfade:** Alle Pfade müssen via `path/filepath` gehandhabt werden, um Kompatibilität zwischen Linux/macOS und Windows sicherzustellen.
- **Taskfile First:** Alle administrativen Aufgaben (Build, Model-Download, Docker) erfolgen ausschließlich über `Taskfile.yml`.

## Styling-Regeln
- **Code Style:** Standard `go fmt` und `golangci-lint` Regeln.
- **Concurrency:** Explizites Handling von Worker-Pools für GPU-Jobs, um Ressourcen-Konflikte zu vermeiden.
- **Error Handling:** Explizite Error-Rückgaben; Gin-Kontext nur in der API-Schicht nutzen.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI
- **Status:** Aktuell
