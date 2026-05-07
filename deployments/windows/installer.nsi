; ─── MLC Upscale — NSIS Installer ─────────────────────────────────────────────
; Requires NSIS 3.x with MUI2 + nsDialogs plugin
; ───────────────────────────────────────────────────────────────────────────────

!ifndef APP_VERSION
  !define APP_VERSION "1.0.4"
!endif

; ─── General ──────────────────────────────────────────────────────────────────

Name               "MLC Upscale"
OutFile            "../../build/mlcupscale-${APP_VERSION}-setup.exe"
InstallDir         "$PROGRAMFILES64\MLC Upscale"
InstallDirRegKey   HKLM "Software\MLC Upscale" "InstallDir"
RequestExecutionLevel admin
Unicode            true
SetCompressor      /SOLID lzma

; ─── MUI2 ─────────────────────────────────────────────────────────────────────

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "FileFunc.nsh"

!define MUI_ABORTWARNING

; ─── Pages ────────────────────────────────────────────────────────────────────

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
Page custom DesktopShortcutPage DesktopShortcutPageLeave
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; ─── Languages ────────────────────────────────────────────────────────────────

!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "German"

; ─── Translated strings ────────────────────────────────────────────────────────

LangString DESKTOP_TITLE      ${LANG_ENGLISH}        "Desktop Shortcut"
LangString DESKTOP_TITLE      ${LANG_GERMAN}         "Desktop-Verknüpfung"

LangString DESKTOP_CHECKBOX   ${LANG_ENGLISH}        "Create a desktop shortcut for the server"
LangString DESKTOP_CHECKBOX   ${LANG_GERMAN}         "Verknüpfung für den Server auf dem Desktop anlegen"

; ─── Desktop shortcut custom page ─────────────────────────────────────────────

Var DesktopShortcut   ; BST_CHECKED (1) = create, BST_UNCHECKED (0) = skip
Var DesktopCheckbox

Function DesktopShortcutPage
  !insertmacro MUI_HEADER_TEXT "$(DESKTOP_TITLE)" ""
  nsDialogs::Create 1018
  Pop $0

  ${NSD_CreateCheckbox} 0 30u 100% 12u "$(DESKTOP_CHECKBOX)"
  Pop $DesktopCheckbox
  ${NSD_SetState} $DesktopCheckbox ${BST_CHECKED}   ; checked by default

  nsDialogs::Show
FunctionEnd

Function DesktopShortcutPageLeave
  ${NSD_GetState} $DesktopCheckbox $DesktopShortcut
FunctionEnd

; ─── Install section ──────────────────────────────────────────────────────────

Section "MLC Upscale" SecMain
  SectionIn RO

  SetOutPath "$INSTDIR"

  ; Core Binaries
  File "/oname=upscale-server.exe" "../../build/upscale-server-windows-amd64.exe"
  File "/oname=upscale-client.exe" "../../build/upscale-client-windows-amd64.exe"
  File "/oname=upscale-tray.exe" "../../build/upscale-tray-windows-amd64.exe"
  
  ; Engine Binary
  File "/oname=realesrgan-ncnn-vulkan.exe" "../../bin/realesrgan-ncnn-vulkan.exe"

  ; Configuration
  SetOutPath "$INSTDIR\config"
  File "/oname=config.yaml" "../../config/config.windows.yaml"

  ; Models
  SetOutPath "$INSTDIR\models"
  File /r "../../models\*.*"

  ; Data Directories
  CreateDirectory "$INSTDIR\data\uploads"
  CreateDirectory "$INSTDIR\data\outputs"

  ; Registry entries
  WriteRegStr   HKLM "Software\MLC Upscale" "InstallDir" "$INSTDIR"
  WriteRegStr   HKLM "Software\MLC Upscale" "Version"    "${APP_VERSION}"

  ; Autostart
  WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Run" "MLCUpscaleTray" "$INSTDIR\upscale-tray.exe"

  WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\MLCUpscale" \
                     "DisplayName"          "MLC Upscale"
  WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\MLCUpscale" \
                     "DisplayVersion"       "${APP_VERSION}"
  WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\MLCUpscale" \
                     "Publisher"            "Michael Lechner"
  WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\MLCUpscale" \
                     "UninstallString"      '"$INSTDIR\uninstall.exe"'
  WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\MLCUpscale" \
                     "DisplayIcon"          "$INSTDIR\upscale-server.exe,0"
  
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Start menu shortcuts
  CreateDirectory "$SMPROGRAMS\MLC Upscale"
  CreateShortcut  "$SMPROGRAMS\MLC Upscale\MLC Upscale Tray.lnk" \
                  "$INSTDIR\upscale-tray.exe" "" "$INSTDIR\upscale-tray.exe" 0
  CreateShortcut  "$SMPROGRAMS\MLC Upscale\MLC Upscale Server.lnk" \
                  "$INSTDIR\upscale-server.exe" "" "$INSTDIR\upscale-server.exe" 0
  CreateShortcut  "$SMPROGRAMS\MLC Upscale\Uninstall MLC Upscale.lnk" \
                  "$INSTDIR\uninstall.exe"

  ; Optional desktop shortcut
  ${If} $DesktopShortcut == ${BST_CHECKED}
    CreateShortcut "$DESKTOP\MLC Upscale Tray.lnk" \
                   "$INSTDIR\upscale-tray.exe" "" "$INSTDIR\upscale-tray.exe" 0
  ${EndIf}

SectionEnd

; ─── Uninstall section ────────────────────────────────────────────────────────

Section "Uninstall"
  Delete "$INSTDIR\upscale-server.exe"
  Delete "$INSTDIR\upscale-client.exe"
  Delete "$INSTDIR\upscale-tray.exe"
  Delete "$INSTDIR\realesrgan-ncnn-vulkan.exe"
  Delete "$INSTDIR\uninstall.exe"
  
  RMDir /r "$INSTDIR\config"
  RMDir /r "$INSTDIR\models"
  RMDir /r "$INSTDIR\data"
  RMDir  "$INSTDIR"

  Delete "$SMPROGRAMS\MLC Upscale\MLC Upscale Tray.lnk"
  Delete "$SMPROGRAMS\MLC Upscale\MLC Upscale Server.lnk"
  Delete "$SMPROGRAMS\MLC Upscale\Uninstall MLC Upscale.lnk"
  RMDir  "$SMPROGRAMS\MLC Upscale"

  Delete "$DESKTOP\MLC Upscale Tray.lnk"

  DeleteRegValue HKLM "Software\Microsoft\Windows\CurrentVersion\Run" "MLCUpscaleTray"
  DeleteRegKey HKLM "Software\MLC Upscale"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\MLCUpscale"
SectionEnd
