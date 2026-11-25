; Bộ cài đặt Digital Shop — Nam Linh Group
; Biên dịch: makensis setup.nsi  (từ thư mục installer, sau khi đã có ..\dist\TapHoaX.exe)
Unicode true
SetCompressor /SOLID lzma
ManifestDPIAware true

!include "MUI2.nsh"
!include "x64.nsh"
!include "WinVer.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"

!define APPNAME "Digital Shop"
!define APPKEY "TapHoaX"
!define COMPANY "Nam Linh Group"
!define VERSION "3.1.2"
!define TERMS_VERSION "1.0"
!define UNINSTKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPKEY}"
!define APPREG "Software\NamLinhGroup\${APPKEY}"
!define DATADIR "$APPDATA\Nam Linh Group\TapHoaX\DuLieu"

Name "${APPNAME}"
Caption "Cài đặt ${APPNAME} ${VERSION} — ${COMPANY}"
OutFile "..\dist\TapHoaX-CaiDat-${VERSION}.exe"
InstallDir ""
InstallDirRegKey HKLM "${APPREG}" "InstallDir"
RequestExecutionLevel admin
BrandingText "${COMPANY}"

VIProductVersion "${VERSION}.0"
VIFileVersion "${VERSION}.0"
VIAddVersionKey /LANG=0 "ProductName" "${APPNAME}"
VIAddVersionKey /LANG=0 "CompanyName" "${COMPANY}"
VIAddVersionKey /LANG=0 "LegalCopyright" "© 2026 ${COMPANY}"
VIAddVersionKey /LANG=0 "FileDescription" "Bộ cài đặt ${APPNAME} - phần mềm bán hàng và quản lý kho"
VIAddVersionKey /LANG=0 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=0 "ProductVersion" "${VERSION}"

!define MUI_ICON "taphoax.ico"
!define MUI_UNICON "taphoax.ico"
!define MUI_WELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "header.bmp"
!define MUI_HEADERIMAGE_RIGHT
!define MUI_ABORTWARNING
!define MUI_ABORTWARNING_TEXT "Bạn có chắc muốn dừng cài đặt ${APPNAME}?"
!define MUI_COMPONENTSPAGE_NODESC

; ---- Trang chào ----
!define MUI_WELCOMEPAGE_TITLE "Cài đặt ${APPNAME} ${VERSION}"
!define MUI_WELCOMEPAGE_TEXT "Phần mềm bán hàng và quản lý kho cho cửa hàng tạp hóa, do ${COMPANY} phát triển.$\r$\n$\r$\nBộ cài sẽ chép phần mềm vào máy, tạo biểu tượng trên màn hình, mở cổng mạng nội bộ cho điện thoại trong tiệm và tạo thư mục lưu dữ liệu riêng.$\r$\n$\r$\nNếu Digital Shop đang chạy, bộ cài sẽ tự tắt rồi cập nhật. Dữ liệu bán hàng được giữ nguyên.$\r$\n$\r$\nBấm Tiến để tiếp tục."
!insertmacro MUI_PAGE_WELCOME

; ---- Điều khoản sử dụng: bắt buộc tích đồng ý ----
!define MUI_PAGE_HEADER_TEXT "Điều khoản sử dụng"
!define MUI_PAGE_HEADER_SUBTEXT "Vui lòng đọc kỹ trước khi cài đặt ${APPNAME}."
!define MUI_LICENSEPAGE_TEXT_TOP "Kéo xuống để đọc toàn bộ điều khoản (phiên bản ${TERMS_VERSION})."
!define MUI_LICENSEPAGE_TEXT_BOTTOM "Bạn cần đồng ý với Điều khoản sử dụng để cài đặt ${APPNAME}."
!define MUI_LICENSEPAGE_CHECKBOX
!define MUI_LICENSEPAGE_CHECKBOX_TEXT "Tôi đã đọc và đồng ý Điều khoản sử dụng"
!insertmacro MUI_PAGE_LICENSE "DieuKhoanSuDung_utf16.txt"

; ---- Tùy chọn ----
!define MUI_PAGE_HEADER_TEXT "Tùy chọn cài đặt"
!define MUI_PAGE_HEADER_SUBTEXT "Chọn những mục muốn cài."
!define MUI_COMPONENTSPAGE_TEXT_TOP "Bỏ chọn mục không cần. Có thể đổi lại bằng cách chạy bộ cài lần nữa."
!insertmacro MUI_PAGE_COMPONENTS

!define MUI_PAGE_HEADER_TEXT "Thư mục cài đặt"
!define MUI_PAGE_HEADER_SUBTEXT "Nơi chép phần mềm. Dữ liệu bán hàng luôn lưu riêng tại ProgramData."
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES

; ---- Hoàn tất ----
!define MUI_FINISHPAGE_TITLE "Đã cài đặt xong ${APPNAME}"
!define MUI_FINISHPAGE_TEXT "Mở ${APPNAME} bằng biểu tượng trên màn hình hoặc trong menu Start → ${COMPANY}.$\r$\n$\r$\nLần đầu mở: đặt tên cửa hàng và mã PIN chủ tiệm."
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Mở ${APPNAME} ngay"
!define MUI_FINISHPAGE_RUN_FUNCTION RunAsUser
!define MUI_FINISHPAGE_SHOWREADME "$INSTDIR\HuongDan.txt"
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Xem hướng dẫn sử dụng"
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!insertmacro MUI_PAGE_FINISH

!define MUI_UNPAGE_CONFIRM_TEXT_TOP "${APPNAME} sẽ được gỡ khỏi máy. Bạn sẽ được hỏi có giữ lại dữ liệu bán hàng không."
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "Vietnamese"

Function .onInit
  ${IfNot} ${AtLeastWin10}
    MessageBox MB_ICONSTOP "${APPNAME} cần Windows 10 hoặc Windows 11. Máy này đang dùng phiên bản Windows cũ hơn."
    Abort
  ${EndIf}
  SetShellVarContext all
  ${If} ${RunningX64}
    SetRegView 64
    ${If} $INSTDIR == ""
      StrCpy $INSTDIR "$PROGRAMFILES64\${COMPANY}\${APPKEY}"
    ${EndIf}
  ${Else}
    ${If} $INSTDIR == ""
      StrCpy $INSTDIR "$PROGRAMFILES\${COMPANY}\${APPKEY}"
    ${EndIf}
  ${EndIf}
FunctionEnd

; Mở phần mềm bằng quyền người dùng thường (không chạy dưới quyền quản trị của bộ cài)
Function RunAsUser
  Exec '"$WINDIR\explorer.exe" "$INSTDIR\TapHoaX.exe"'
FunctionEnd

Section "${APPNAME} (bắt buộc)" SecMain
  SectionIn RO
  SetShellVarContext all
  DetailPrint "Tắt ${APPNAME} nếu đang chạy..."
  nsExec::Exec 'taskkill /F /IM TapHoaX.exe'
  Pop $0
  Sleep 800

  SetOutPath "$INSTDIR"
  ${If} ${RunningX64}
    File "/oname=TapHoaX.exe" "..\dist\TapHoaX.exe"
  ${Else}
    File "/oname=TapHoaX.exe" "..\dist\TapHoaX-32bit.exe"
  ${EndIf}
  File "..\dist\HuongDan.txt"
  File "..\dist\DieuKhoanSuDung.txt"
  File "..\dist\GiayPhepBenThuBa.txt"
  File "taphoax.ico"

  ; Thư mục dữ liệu chung cho mọi tài khoản Windows ($APPDATA = C:\ProgramData khi SetShellVarContext all), người dùng thường được ghi
  DetailPrint "Tạo thư mục dữ liệu ${DATADIR}"
  CreateDirectory "${DATADIR}"
  nsExec::Exec 'icacls "$APPDATA\Nam Linh Group\TapHoaX" /grant *S-1-5-32-545:(OI)(CI)M /T /Q'
  Pop $0
  nsExec::Exec 'icacls "$INSTDIR" /grant *S-1-5-32-545:(OI)(CI)M /T /Q'
  Pop $0 ; cho phép tự cập nhật phần mềm
  FileOpen $1 "$INSTDIR\datadir.txt" w
  FileWrite $1 "${DATADIR}"
  FileClose $1

  ; Cho điện thoại trong mạng nội bộ kết nối tới phần mềm
  DetailPrint "Mở tường lửa cho mạng nội bộ..."
  nsExec::Exec 'netsh advfirewall firewall delete rule name="TapHoaX"'
  Pop $0
  nsExec::Exec 'netsh advfirewall firewall add rule name="TapHoaX" dir=in action=allow program="$INSTDIR\TapHoaX.exe" enable=yes profile=any remoteip=localsubnet'
  Pop $0

  ; Menu Start
  CreateDirectory "$SMPROGRAMS\${COMPANY}"
  CreateShortcut "$SMPROGRAMS\${COMPANY}\${APPNAME}.lnk" "$INSTDIR\TapHoaX.exe" "" "$INSTDIR\taphoax.ico" 0 SW_SHOWNORMAL "" "Phần mềm bán hàng ${APPNAME}"
  CreateShortcut "$SMPROGRAMS\${COMPANY}\Hướng dẫn ${APPNAME}.lnk" "$INSTDIR\HuongDan.txt"
  CreateShortcut "$SMPROGRAMS\${COMPANY}\Gỡ cài đặt ${APPNAME}.lnk" "$INSTDIR\GoCaiDat.exe"

  ; Gỡ cài đặt trong Settings → Apps của Windows
  WriteUninstaller "$INSTDIR\GoCaiDat.exe"
  WriteRegStr HKLM "${UNINSTKEY}" "DisplayName" "${APPNAME}"
  WriteRegStr HKLM "${UNINSTKEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINSTKEY}" "Publisher" "${COMPANY}"
  WriteRegStr HKLM "${UNINSTKEY}" "DisplayIcon" "$INSTDIR\taphoax.ico"
  WriteRegStr HKLM "${UNINSTKEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "${UNINSTKEY}" "UninstallString" '"$INSTDIR\GoCaiDat.exe"'
  WriteRegStr HKLM "${UNINSTKEY}" "QuietUninstallString" '"$INSTDIR\GoCaiDat.exe" /S'
  WriteRegDWORD HKLM "${UNINSTKEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINSTKEY}" "NoRepair" 1
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "${UNINSTKEY}" "EstimatedSize" "$0"

  ; Ghi nhận việc đồng ý điều khoản khi cài
  ${GetTime} "" "L" $2 $3 $4 $5 $6 $7 $8
  WriteRegStr HKLM "${APPREG}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "${APPREG}" "DataDir" "${DATADIR}"
  WriteRegStr HKLM "${APPREG}" "TermsAccepted" "${TERMS_VERSION}"
  WriteRegStr HKLM "${APPREG}" "TermsAcceptedAt" "$4-$3-$2 $6:$7:$8"
  ReadEnvStr $9 USERNAME
  WriteRegStr HKLM "${APPREG}" "TermsAcceptedBy" "$9"
SectionEnd

Section "Biểu tượng trên màn hình" SecDesktop
  SetShellVarContext all
  CreateShortcut "$DESKTOP\${APPNAME}.lnk" "$INSTDIR\TapHoaX.exe" "" "$INSTDIR\taphoax.ico" 0 SW_SHOWNORMAL "" "Phần mềm bán hàng ${APPNAME}"
SectionEnd

Section "Tự mở khi bật máy" SecStartup
  SetShellVarContext all
  CreateShortcut "$SMSTARTUP\${APPNAME}.lnk" "$INSTDIR\TapHoaX.exe" "" "$INSTDIR\taphoax.ico" 0 SW_SHOWNORMAL "" "Phần mềm bán hàng ${APPNAME}"
SectionEnd

Section "Uninstall"
  SetShellVarContext all
  SetOutPath "$TEMP" ; rời thư mục cài để xóa được
  ${If} ${RunningX64}
    SetRegView 64
  ${EndIf}
  nsExec::Exec 'taskkill /F /IM TapHoaX.exe'
  Pop $0
  Sleep 800
  nsExec::Exec 'netsh advfirewall firewall delete rule name="TapHoaX"'
  Pop $0
  Delete "$INSTDIR\TapHoaX.exe"
  Delete "$INSTDIR\HuongDan.txt"
  Delete "$INSTDIR\DieuKhoanSuDung.txt"
  Delete "$INSTDIR\GiayPhepBenThuBa.txt"
  Delete "$INSTDIR\taphoax.ico"
  Delete "$INSTDIR\datadir.txt"
  Delete "$INSTDIR\GoCaiDat.exe"
  RMDir "$INSTDIR"
  RMDir "$PROGRAMFILES64\${COMPANY}"
  RMDir "$PROGRAMFILES\${COMPANY}"
  Delete "$DESKTOP\${APPNAME}.lnk"
  Delete "$SMSTARTUP\${APPNAME}.lnk"
  Delete "$SMPROGRAMS\${COMPANY}\${APPNAME}.lnk"
  Delete "$SMPROGRAMS\${COMPANY}\Hướng dẫn ${APPNAME}.lnk"
  Delete "$SMPROGRAMS\${COMPANY}\Gỡ cài đặt ${APPNAME}.lnk"
  RMDir "$SMPROGRAMS\${COMPANY}"
  DeleteRegKey HKLM "${UNINSTKEY}"

  MessageBox MB_YESNO|MB_ICONQUESTION|MB_DEFBUTTON2 "Xóa luôn toàn bộ dữ liệu bán hàng (hàng hóa, hóa đơn, sổ nợ, bản sao lưu)?$\r$\n$\r$\nChọn KHÔNG để giữ lại dữ liệu (khuyên dùng). Cài lại sau sẽ dùng tiếp được." /SD IDNO IDNO keep
    RMDir /r "$APPDATA\Nam Linh Group\TapHoaX"
    RMDir "$APPDATA\Nam Linh Group"
    DeleteRegKey HKLM "${APPREG}"
    Goto done
  keep:
    DetailPrint "Đã giữ lại dữ liệu tại $APPDATA\Nam Linh Group\TapHoaX"
  done:
SectionEnd
