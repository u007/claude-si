: << 'CMDBLOCK'
@echo off
setlocal
if /I not "%OS%"=="Windows_NT" (
  echo okf: unsupported operating system: %OS% 1>&2
  exit /b 1
)
set "ROOT=%~dp0.."
set "OS_NAME=windows"
set "ARCH=%PROCESSOR_ARCHITECTURE%"
if /I "%ARCH%"=="AMD64" set "ARCH=amd64"
if /I "%ARCH%"=="ARM64" set "ARCH=arm64"
if /I not "%ARCH%"=="amd64" if /I not "%ARCH%"=="arm64" (
  echo okf: unsupported architecture: %PROCESSOR_ARCHITECTURE% 1>&2
  exit /b 1
)
set "BIN=%ROOT%\bin\%OS_NAME%\%ARCH%\okf.exe"
if not exist "%BIN%" (
  echo okf: missing binary for %OS_NAME%/%ARCH%: "%BIN%" 1>&2
  exit /b 1
)
"%BIN%" %*
exit /b %ERRORLEVEL%
CMDBLOCK

set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  MINGW*|MSYS*|CYGWIN*) os=windows ;;
  *)
    echo "okf: unsupported operating system: $(uname -s)" >&2
    exit 1
    ;;
esac
case "$(uname -m)" in
  x86_64|amd64|AMD64) arch=amd64 ;;
  arm64|aarch64|ARM64) arch=arm64 ;;
  *)
    echo "okf: unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac
suffix=
if [ "$os" = windows ]; then
  suffix=.exe
fi
bin="$root/bin/$os/$arch/okf$suffix"
if [ ! -f "$bin" ]; then
  echo "okf: missing binary for $os/$arch: $bin" >&2
  exit 1
fi
exec "$bin" "$@"
