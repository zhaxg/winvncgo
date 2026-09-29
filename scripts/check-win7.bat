@echo off
setlocal enabledelayedexpansion
rem ============================================================
rem  WinVNC Go - one-click Windows 7 startup diagnostic
rem  Output file: diag-report.txt (in this folder)
rem ============================================================
set "DIR=%~dp0"
cd /d "%DIR%"
set "REPORT=%DIR%diag-report.txt"

echo ============================================
echo   WinVNC Go startup diagnostic
echo   Report file: %REPORT%
echo ============================================

> "%REPORT%" (
  echo WinVNC Go diagnostic report
  echo Time: %date% %time%
  echo Folder: %DIR%
)

>> "%REPORT%" (
  echo.
  echo ===== 1. System =====
  ver
  echo Processor arch : %PROCESSOR_ARCHITECTURE%
  echo Arch64 override: %PROCESSOR_ARCHITEW6432%
  if defined PROCESSOR_ARCHITEW6432 (
    echo NOTE: 32-bit process on 64-bit Windows
  ) else if "%PROCESSOR_ARCHITECTURE%"=="x86" (
    echo NOTE: 32-bit Windows - winvncgo.exe is 64-bit and CANNOT start here.
  )
)

rem ---- run win7diag (console program; shows Go runtime fatal errors on screen)
echo.
echo --- Step 2: opening win7diag in a command window ---
echo     (read that window; close it when done)
if exist "%DIR%win7diag.exe" (
  start "win7diag - Go runtime check" /k ""%DIR%win7diag.exe" -nopause"
) else (
  echo win7diag.exe NOT FOUND in %DIR%
  >> "%REPORT%" echo win7diag.exe NOT FOUND
)

rem ---- run the app with a console so fatal errors become visible
echo.
echo --- Step 3: opening winvncgo-console.exe in a command window ---
echo     (if a program window appears, the app starts - close it when done;
echo      if text like "fatal error: ..." appears, that is the answer)
if exist "%DIR%winvncgo-console.exe" (
  start "winvncgo-console - app startup" /k ""%DIR%winvncgo-console.exe""
) else (
  echo winvncgo-console.exe NOT FOUND in %DIR%
  >> "%REPORT%" echo winvncgo-console.exe NOT FOUND
)

rem ---- also try the GUI exe and record its exit code
echo.
echo --- Step 4: launching winvncgo.exe and recording exit code ---
set "EC=not-run"
if exist "%DIR%winvncgo.exe" (
  start "winvncgo" /wait "%DIR%winvncgo.exe"
  set "EC=!errorlevel!"
) else (
  echo winvncgo.exe NOT FOUND
)
echo Exit code: %EC%

>> "%REPORT%" (
  echo.
  echo ===== 5. winvncgo.exe exit code =====
  echo exit code: %EC%
)

rem ---- decode common NTSTATUS values
>> "%REPORT%" (
  if "%EC%"=="-1073741515" echo   = 0xC0000135 STATUS_DLL_NOT_FOUND: a required DLL is missing
  if "%EC%"=="3221225781"  echo   = 0xC0000135 STATUS_DLL_NOT_FOUND: a required DLL is missing
  if "%EC%"=="-1073741502" echo   = 0xC0000142 STATUS_DLL_INIT_FAILED: DLL initialization failed
  if "%EC%"=="3221225794"  echo   = 0xC0000142 STATUS_DLL_INIT_FAILED: DLL initialization failed
  if "%EC%"=="-1073741512" echo   = 0xC0000138 STATUS_ORDINAL_NOT_FOUND: an export is missing
  if "%EC%"=="3221225784"  echo   = 0xC0000138 STATUS_ORDINAL_NOT_FOUND: an export is missing
  if "%EC%"=="-1073741819" echo   = 0xC0000005 ACCESS_VIOLATION: crashed
  if "%EC%"=="3221225477"  echo   = 0xC0000005 ACCESS_VIOLATION: crashed
  if "%EC%"=="-1073740791" echo   = 0xC0000409 STATUS_FAIL_FAST / stack corruption
  if "%EC%"=="3221226505"  echo   = 0xC0000409 STATUS_FAIL_FAST / stack corruption
  if "%EC%"=="1"           echo   = clean exit with error (log.Fatal path - see winvncgo.log)
  if "%EC%"=="0"           echo   = normal exit
)

rem ---- log files, both possible locations
>> "%REPORT%" (
  echo.
  echo ===== 6. Log files =====
)
call :dumplog "%DIR%winvncgo.log"
call :dumplog "%TEMP%\winvncgo.log"
call :dumplog "%DIR%win7diag.log"

rem ---- recent application errors from event log
>> "%REPORT%" (
  echo.
  echo ===== 7. Windows event log (last 8 application errors) =====
)
wevtutil qe Application /c:8 /f:text /rd:true /q:"*[System[(Level=1 or Level=2)]]" >> "%REPORT%" 2>&1

echo.
echo ============================================
echo   Done. Opening diag-report.txt ...
echo   Please send this file back.
echo ============================================
notepad "%REPORT%"
exit /b 0

:dumplog
>> "%REPORT%" (
  echo.
  echo --- file: %~1 ---
)
if exist "%~1" (
  type "%~1" >> "%REPORT%"
) else (
  echo (does not exist) >> "%REPORT%"
)
exit /b 0
