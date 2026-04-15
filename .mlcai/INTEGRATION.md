# 🤖 AI & Integration Context: MLC Upscale Service

## 1. Identität & Zweck
- **Kernaufgabe:** Hochperformante REST API für AI-basiertes Image Upscaling mit Real-ESRGAN (ncnn-vulkan). Optimiert für die Verarbeitung massiver Bilder (bis zu 10.000px+) durch asynchrone Workflows.
- **Technischer Stack:** Go (Gin Framework), Real-ESRGAN (C++/Vulkan Binaries).
- **Hoster/Infrastruktur:** Docker (Linux), Native macOS (Apple Silicon), Windows (WSL2/Native).

## 2. Die "Nachbarschaft" (System-Kontext)
- **Upstream (Wovon hänge ich ab?):**
  - **Real-ESRGAN** -> [GitHub](https://github.com/xinntao/Real-ESRGAN) -> Liefert die zugrundeliegenden Modelle und die `ncnn-vulkan` Binaries.
- **Downstream (Wer nutzt mich?):**
  - **Mac Studio / Desktop Clients:** Generieren Bilder (z.B. via Ollama/Stable Diffusion) und nutzen diesen Service für das finale Upscaling auf Poster-Größe.
  - **Automatisierte Pipelines:** Nutzen die API für Batch-Verarbeitung von Bildmaterial.

## 3. Schnittstellen-Vertrag
- **Primäre API:** REST (Asynchron)
- **Auth-Mechanismus:** Bearer Auth / API-Key (konfigurierbar).
- **Workflow:**
  1. `POST /upscale` -> Upload Image & Parameter -> Erhält `job_id`.
  2. `GET /status/{job_id}` -> Pollen bis Status `completed`.
  3. `GET /download/{job_id}` -> Download des Resultats.
- **API-Doku-Link:** [docs/openapi.yaml](docs/openapi.yaml) oder [docs/API_GUIDE.md](docs/API_GUIDE.md).

## 4. Leitplanken & Regeln
- **Asynchronität:** Alle Upscale-Operationen MÜSSEN asynchron sein, um Timeouts bei großen Bildern zu vermeiden.
- **Resource Management:** `tile_size` sollte bei VRAM-Mangel (insb. in Docker) auf Werte wie 400-800 gesetzt werden.
- **Cleanup:** Temporäre Dateien in `data/uploads` und `data/outputs` werden automatisch nach Ablauf einer konfigurierbaren Frist gelöscht.

## 5. Aktueller Fokus (Status)
- **Status:** Beta.
- **Bekannte Probleme:** 4x Upscaling kann bei sehr komplexen Texturen zu Artefakten führen; 2x/3x sind stabiler.
- **Nächste Schritte:** Integration von weiteren ONNX-basierten Modellen und verbesserte Metriken für GPU-Auslastung.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI
- **Status:** Aktuell
