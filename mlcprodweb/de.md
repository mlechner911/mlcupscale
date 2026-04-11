# Hochleistungs-KI-Bild-Upscaling

MLCUpscale ist eine Hochleistungs-REST-API, die für eine spezifische Aufgabe entwickelt wurde: das Upscaling riesiger Bilder mithilfe modernster KI-Modelle. Während viele Tools bereits bei Bildern mit wenigen Megapixeln Probleme haben, ist MLCUpscale darauf ausgelegt, Dateien mit einer Größe von 10.000x10.000 Pixeln und mehr zu verarbeiten, indem es fortschrittliche Tiling-Strategien und GPU-Beschleunigung nutzt.

## Über die Grenzen von Consumer-Hardware hinaus

Durch das Auslagern des rechenintensiven Upscaling-Prozesses auf einen dedizierten Server können Clients professionelles Bildmaterial ohne lokale Hardware-Einschränkungen verarbeiten. MLCUpscale nutzt Vulkan-basierte Hardwarebeschleunigung und ist somit mit einer Vielzahl von GPUs unter Linux und macOS (einschließlich Apple Silicon) kompatibel.

## Asynchroner Workflow

1. **Senden**: Senden Sie eine POST-Anfrage mit Ihrem Bild und dem gewünschten Skalierungsfaktor (2x, 3x oder 4x).
2. **Verfolgen**: Rufen Sie den Status-Endpunkt auf, um den Fortschritt in Echtzeit zu verfolgen.
3. **Herunterladen**: Sobald der Vorgang abgeschlossen ist, rufen Sie das hochauflösende Ergebnis ab.

## API Schnellstart

Einen neuen Upscaling-Job einreichen:

```bash
curl -X POST http://api.mlcgo.eu/v1/upscale \
  -F "image=@photo.jpg" \
  -F "scale=4" \
  -F "model_name=realesrgan-x4plus"
```

Status prüfen:

```bash
curl http://api.mlcgo.eu/v1/status/IHRE_JOB_ID
```

Ergebnis herunterladen:

```bash
curl -O http://api.mlcgo.eu/v1/download/IHRE_JOB_ID
```
