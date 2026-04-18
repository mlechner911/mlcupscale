# 🔌 API Contract: MLC Upscale — Image Upscaling API

## 1. Bereitgestellte Endpunkte (Exposed)
Der Service bietet eine asynchrone REST-API zur Hochskalierung von Bildern.

### REST API (v1)
- **Basis-URL:** `/api/v1` (konfigurierbar via `api_prefix`)
- **Auth:** Optionaler Bearer Token im `Authorization` Header oder `X-Auth-Token` Header.
- **Wichtige Endpunkte:**
  - `POST /upscale`: Reicht einen neuen Upscaling-Job ein (Multipart Form-Data).
    - Parameter: `image` (file), `scale` (2, 3, 4), `model_name` (string), `tile_size` (int), `format` (png, jpg, webp).
    - Response: `202 Accepted` mit `job_id`.
  - `GET /status/{job_id}`: Prüft den Fortschritt (0-100) und Status eines Jobs.
  - `GET /download/{job_id}`: Lädt das fertige Bild herunter (nur bei Status `completed`).
  - `POST /cancel/{job_id}`: Bricht einen laufenden oder wartenden Job ab.
  - `GET /models`: Listet verfügbare KI-Modelle und deren Beschreibungen.
  - `GET /health`: Health-Check (Status, Version, Serverzeit).
  - `GET /docs`: Interaktive Swagger-Dokumentation (sofern in Config aktiviert).
- **Fehlerformat:**
  ```json
  { "success": false, "error": "Fehlermeldung" }
  ```

### Dokumentation
- **Swagger UI:** Erreichbar unter `/api/v1/docs` (lokal: `http://localhost:8089/api/v1/docs`).
- **Spezifikation:** `docs/openapi.yaml`.

-----

## 2. Konsumierte APIs (Consumed)
Dieses Projekt ist ein eigenständiger Service und konsumiert keine externen Web-APIs. Es nutzt jedoch externe Binaries:

### Extern: Real-ESRGAN (ncnn-vulkan)
- **Zweck:** Eigentliche Bildverarbeitung via GPU (Vulkan) oder CPU.
- **Integration:** Aufruf über das Betriebssystem (`exec.Command`) mit Fortschrittsanalyse über `stderr`.

-----

## 3. Globale Konventionen
- **Datumsformat:** Immer ISO 8601 (UTC), z.B. `2026-04-18T12:00:00Z`.
- **Job IDs:** Einzigartige Strings basierend auf UnixNano (z.B. `1713430000000000000`).
- **Dateispeicherung:** Uploads und Ergebnisse werden lokal in `data/` gespeichert und nach 15 Minuten automatisch gelöscht.
- **Paginierung:** Derzeit nicht implementiert, da Job-Listen nicht exponiert werden.
- **Rate-Limiting:** Konfigurierbar via `LimitsConfig` (Standard: 100 Requests/Min).
- **Async:** Standard-Workflow ist asynchron: Upload -> Poll -> Download.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-18
- **Aktualisiert von:** gemini-2.0-flash-exp
- **Status:** Aktuell
