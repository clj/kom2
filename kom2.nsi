Name "KiCad Odbc Middleware (Inventree)"

CRCCheck On

!define PROD_NAME "KiCad ODBC Middleware"
!define PROD_NAME0 "KiCad ODBC Middleware"
!ifndef DLL
  !define DLL "kom2.dll"
!endif

!include "MUI.nsh"
!include "Sections.nsh"

OutFile "kom2-installer.exe"

ShowInstDetails show
AutoCloseWindow false

; Function IndexOf
; Exch $R0
; Exch
; Exch $R1
; Push $R2
; Push $R3
 
;  StrCpy $R3 $R0
;  StrCpy $R0 -1
;  IntOp $R0 $R0 + 1
;   StrCpy $R2 $R3 1 $R0
;   StrCmp $R2 "" +2
;   StrCmp $R2 $R1 +2 -3
 
;  StrCpy $R0 -1
 
; Pop $R3
; Pop $R2
; Pop $R1
; Exch $R0
; FunctionEnd
 
; !macro IndexOf Var Str Char
; Push "${Char}"
; Push "${Str}"
;  Call IndexOf
; Pop "${Var}"
; !macroend
; !define IndexOf "!insertmacro IndexOf"
 
Function RIndexOf
Exch $R0
Exch
Exch $R1
Push $R2
Push $R3
 
 StrCpy $R3 $R0
 StrCpy $R0 0
 IntOp $R0 $R0 + 1
  StrCpy $R2 $R3 1 -$R0
  StrCmp $R2 "" +2
  StrCmp $R2 $R1 +2 -3
 
 StrCpy $R0 -1
 
Pop $R3
Pop $R2
Pop $R1
Exch $R0
FunctionEnd
 
!macro RIndexOf Var Str Char
Push "${Char}"
Push "${Str}"
 Call RIndexOf
Pop "${Var}"
!macroend
!define RIndexOf "!insertmacro RIndexOf"

Function CheckAlreadyInstalled
    ReadRegStr $0 HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PROD_NAME0}" \
        "UninstallString"

    StrCmp $0 "" INSTALL

    MessageBox MB_YESNO "KiCad ODBC Middleware seems to  \
                          already be installed. Would you like to \
                          uninstall it first?" \
                          IDYES UNINSTALL IDNO INSTALL

    UNINSTALL:
     ${RIndexOf} $R0 $0 "\"
     StrCpy $1 $0 -$R0
     ExecWait '"$0" _?=$1'

    INSTALL:
FunctionEnd

 Function .onInit
   Call CheckAlreadyInstalled
 FunctionEnd

InstallDir "$PROGRAMFILES64\${PROD_NAME0}"

!define MUI_WELCOMEPAGE_TITLE "KiCad ODBC Middleware Installation"
!define MUI_WELCOMEPAGE_TEXT "This program will guide you through the \
installation of the KiCad ODBC Middleware Driver.\r\n\r\n$_CLICK"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES

!define MUI_FINISHPAGE_TITLE "KiCad ODBC Middleware Installation"
!define MUI_FINISHPAGE_TEXT "The installation of the KiCad ODBC Middleware Driver is complete.\
\r\n\r\n$_CLICK"

!define MUI_FINISHPAGE_NOAUTOCLOSE

!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"


Section "-Main (required)" InstallationInfo
; Add files
 SetOutPath "$INSTDIR"
  File "inst.exe"
  File "${DLL}"
  File "LICENSE"
  File "README.md"

 SetOutPath "$SMPROGRAMS\${PROD_NAME0}"
 CreateShortCut "$SMPROGRAMS\${PROD_NAME0}\Uninstall.lnk" \
   "$INSTDIR\Uninstall.exe"

; Write uninstall information to the registry
 WriteRegStr HKLM \
  "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PROD_NAME0}" \
  "DisplayName" "${PROD_NAME} (remove only)"
 WriteRegStr HKLM \
  "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PROD_NAME0}" \
  "UninstallString" "$INSTDIR\Uninstall.exe"

 SetOutPath "$INSTDIR"
 WriteUninstaller "$INSTDIR\Uninstall.exe"

 DetailPrint "Adding ODBC driver"
 nsExec::ExecToLog '"$INSTDIR\inst.exe" install --dll=${DLL}'
SectionEnd


Section "Uninstall"

DetailPrint "Removing ODBC driver"
nsExec::ExecToLog '"$INSTDIR\inst.exe" uninstall --dll=${DLL}'
 
; Delete Files 
RMDir /r "$INSTDIR\*" 
RMDir /r "$INSTDIR\*.*" 
 
; Remove the installation directory
RMDir /r "$INSTDIR"

; Remove start menu/program files subdirectory

RMDir /r "$SMPROGRAMS\${PROD_NAME0}"
  
; Delete Uninstaller And Unistall Registry Entries
DeleteRegKey HKEY_LOCAL_MACHINE "SOFTWARE\${PROD_NAME0}"
DeleteRegKey HKEY_LOCAL_MACHINE \
    "SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\${PROD_NAME0}"
  
SectionEnd
