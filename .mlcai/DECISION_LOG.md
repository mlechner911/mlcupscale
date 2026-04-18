# 📜 Decision Log (ADR): MLC Upscale

Dieses Dokument listet fundamentale Entscheidungen auf, die im Projekt getroffen wurden.

## 2026-04-15: Asynchrone Job-Verarbeitung
- **Kontext:** Upscaling von Bildern (insbesondere sehr großen wie 10k x 10k) kann Minuten dauern und blockiert HTTP-Verbindungen.
- **Entscheidung:** Wir nutzen ein asynchrones Modell (Submit -> Poll Status -> Download).
- **Grund:** Bessere Benutzererfahrung, Vermeidung von Timeouts, Entkopplung von API und Worker.
- **Konsequenz:** Client muss Polling implementieren; Backend benötigt Job-Queue und Status-Tracking.

---

## 2026-04-15: Nutzung externer Binaries (ncnn-vulkan)
- **Kontext:** KI-Modelle in Go nativ (z.B. via ONNX Runtime Bindings) sind komplex zu implementieren und schwer hardwareübergreifend zu optimieren.
- **Entscheidung:** Wir nutzen die optimierten `realesrgan-ncnn-vulkan` (C++/Vulkan) Binaries.
- **Grund:** Native GPU-Beschleunigung (Vulkan/Metal) ohne komplexe Bindings, einfache Wartbarkeit, stabile Performance.
- **Konsequenz:** Deployment benötigt die entsprechenden Binaries im `bin/`-Ordner oder Bundle.

---

## 2026-04-15: Erzwingen einer Tiling-Strategie
- **Kontext:** Sehr große Bilder führen bei GPUs mit wenig VRAM oder auf Apple Silicon (CoreML Graph Size Limits) zu OOM-Abstürzen.
- **Entscheidung:** Wir erzwingen eine Tile-Size von 512 (Standard) für alle Jobs.
- **Grund:** Stabilität bei massiven Bildern (4K -> 16K) sicherstellen, OOM-Fehler minimieren.
- **Konsequenz:** Verarbeitung dauert etwas länger (Tiling-Overhead), ist aber deutlich robuster.

---

## 2026-04-15: Lokaler File-Speicher mit Cleanup
- **Kontext:** Hochskalierte Bilder sind massiv (viele hundert MB). Cloud-Speicher (S3) verursacht hohe Latenz/Kosten.
- **Entscheidung:** Wir speichern Uploads und Ergebnisse lokal in `data/`.
- **Grund:** Maximale Performance beim Lesen/Schreiben, keine Abhängigkeit von Cloud-Anbietern.
- **Konsequenz:** Ein Cleanup-Dienst löscht Dateien automatisch nach 15 Minuten (TTL) oder nach erfolgreichem Download, um Plattenplatz zu sparen.

---

## 2026-04-15: Gin-Framework für die API
- **Kontext:** Entscheidung zwischen Standard `net/http`, Echo oder Gin.
- **Entscheidung:** Wir nutzen Gin.
- **Grund:** Exzellente Performance, eingebaute Middleware für JSON-Bindung und Logging, weite Verbreitung.
- **Konsequenz:** Idiomatischer Go-API Code mit Gin-Contet.

---

## 2026-04-15: Unterstützung für Python-ONNX (Experimentell)
- **Kontext:** Bedarf an CUDA/TensorRT Support für NVIDIA-GPUs, den ncnn-vulkan nicht optimal abdeckt.
- **Entscheidung:** Einführung eines experimentellen Python-Wrappers für ONNX Runtime.
- **Grund:** Flexibilität für NVIDIA-Nutzer und bessere Portabilität der Modelle.
- **Konsequenz:** Zusätzliche Python-Abhängigkeiten erforderlich (`requirements.txt` in `cmd/python-onnx/`).

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-18
- **Aktualisiert von:** gemini-2.0-flash-exp
- **Status:** Aktuell
