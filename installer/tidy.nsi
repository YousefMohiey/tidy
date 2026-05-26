; tidy Windows Installer - NSIS ModernUI 2
; User-scope installation (no admin/UAC required)

!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "WordFunc.nsh"

!ifndef PRODUCT_VERSION
  !define PRODUCT_VERSION "0.0.0"
!endif
!ifndef PRODUCT_VERSION_NUMERIC
  !define PRODUCT_VERSION_NUMERIC "0.0.0.0"
!endif
!ifndef PRODUCT_INSTALLER_NAME
  !define PRODUCT_INSTALLER_NAME "tidy-Setup"
!endif

!define PRODUCT_NAME "tidy"
!define PRODUCT_PUBLISHER "YousefMohiey"
!define PRODUCT_WEB_SITE "https://github.com/YousefMohiey/tidy"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\tidy"
!define PRODUCT_EXE "tidy.exe"

; --- General attributes ---
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "../dist/${PRODUCT_INSTALLER_NAME}.exe"
InstallDir "$LOCALAPPDATA\Programs\tidy"
InstallDirRegKey HKCU "${PRODUCT_UNINST_KEY}" "InstallLocation"
RequestExecutionLevel user
SetCompressor /SOLID lzma
ShowInstDetails nevershow
ShowUnInstDetails nevershow

; --- ModernUI 2 configuration ---
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; Installer pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "Launch tidy"
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Language
!insertmacro MUI_LANGUAGE "English"

; --- Version information ---
VIProductVersion "${PRODUCT_VERSION_NUMERIC}"
VIAddVersionKey "ProductName" "${PRODUCT_NAME}"
VIAddVersionKey "CompanyName" "${PRODUCT_PUBLISHER}"
VIAddVersionKey "LegalCopyright" "Copyright (c) ${PRODUCT_PUBLISHER}"
VIAddVersionKey "FileDescription" "${PRODUCT_NAME} Installer"
VIAddVersionKey "FileVersion" "${PRODUCT_VERSION}"
VIAddVersionKey "ProductVersion" "${PRODUCT_VERSION}"

; --- Installer sections ---
Section "Install" SecInstall
  SectionIn RO
  SetOutPath "$INSTDIR"

  ; Install the main binary (copied to installer/ by build-installer.sh)
  File "${PRODUCT_EXE}"

  ; Install license
  File "..\LICENSE"

  ; Create uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Get installed size
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0

  ; Register in Add/Remove Programs (HKCU - user scope)
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXE}"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegDWORD HKCU "${PRODUCT_UNINST_KEY}" "EstimatedSize" "$0"
  WriteRegDWORD HKCU "${PRODUCT_UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${PRODUCT_UNINST_KEY}" "NoRepair" 1

  ; Add install directory to user PATH
  Call AddToPath

  ; Create Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\${PRODUCT_EXE}"
  CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall.lnk" "$INSTDIR\uninstall.exe"

  ; Strip Mark-of-the-Web (Zone.Identifier ADS) to prevent SmartScreen on installed files
  nsExec::ExecToLog 'powershell -NoProfile -ExecutionPolicy Bypass -Command "Unblock-File -LiteralPath \"$INSTDIR\${PRODUCT_EXE}\""'
  nsExec::ExecToLog 'powershell -NoProfile -ExecutionPolicy Bypass -Command "Unblock-File -LiteralPath \"$INSTDIR\uninstall.exe\""'
SectionEnd

; --- Uninstaller section ---
Section "Uninstall"
  ; Remove files
  Delete "$INSTDIR\${PRODUCT_EXE}"
  Delete "$INSTDIR\uninstall.exe"
  Delete "$INSTDIR\LICENSE"

  ; Remove install directory
  RMDir "$INSTDIR"

  ; Remove Start Menu shortcuts
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall.lnk"
  RMDir "$SMPROGRAMS\${PRODUCT_NAME}"

  ; Remove from user PATH
  Call un.RemoveFromPath

  ; Remove registry key
  DeleteRegKey HKCU "${PRODUCT_UNINST_KEY}"
SectionEnd

Function AddToPath
  ReadRegStr $0 HKCU "Environment" "Path"
  ${If} $0 != ""
    StrCpy $1 ";$0;"
    ${WordFind} "$1" ";$INSTDIR;" "E+1" $2
    ${IfNot} ${Errors}
      Goto done
    ${EndIf}
    StrCpy $0 "$0;$INSTDIR"
  ${Else}
    StrCpy $0 "$INSTDIR"
  ${EndIf}
  WriteRegExpandStr HKCU "Environment" "Path" "$0"
  SendMessage 0xFFFF 0x001A 0 "STR:Environment" /TIMEOUT=5000
done:
FunctionEnd

Function un.RemoveFromPath
  ReadRegStr $0 HKCU "Environment" "Path"
  ${If} $0 == ""
    Goto done
  ${EndIf}
  ${WordReplace} $0 ";$INSTDIR" "" "+*" $1
  ${If} $1 != $0
    StrCpy $0 $1
    Goto write
  ${EndIf}
  ${WordReplace} $0 "$INSTDIR;" "" "+*" $1
  ${If} $1 != $0
    StrCpy $0 $1
    Goto write
  ${EndIf}
  ${If} $0 == "$INSTDIR"
    StrCpy $0 ""
  ${EndIf}
write:
  WriteRegExpandStr HKCU "Environment" "Path" "$0"
  SendMessage 0xFFFF 0x001A 0 "STR:Environment" /TIMEOUT=5000
done:
FunctionEnd
