@echo off
setlocal enabledelayedexpansion

rem ===========================================================================
rem NOTE: keep this file pure ASCII. cmd.exe reads .cmd in the system ANSI code
rem page, so UTF-8 non-ASCII comments corrupt parsing on CJK Windows.
rem
rem Generator versions must match what produced the checked-in *.pb.go, or every
rem file in the repo flips and buries the real diff. Known differences:
rem   protoc-gen-go-grpc 1.5.x emits status.Errorf(...), 1.6.0 emits status.Error(...)
rem   protoc-gen-go 1.36.8 vs 1.36.11 render line comments differently (//// vs // ///)
rem Do not regenerate with a different version and commit the result.
rem
rem   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0
rem   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
rem ===========================================================================
set "WANT_GRPC=1.6.0"
set "WANT_GO=v1.36.11"

set "HAVE_GRPC="
for /f "tokens=2" %%v in ('protoc-gen-go-grpc --version 2^>^&1') do set "HAVE_GRPC=%%v"
if not "!HAVE_GRPC!"=="%WANT_GRPC%" (
    echo [gen] protoc-gen-go-grpc version mismatch: have "!HAVE_GRPC!", want "%WANT_GRPC%"
    echo [gen]   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v%WANT_GRPC%
    exit /b 1
)

set "HAVE_GO="
for /f "tokens=2" %%v in ('protoc-gen-go --version 2^>^&1') do set "HAVE_GO=%%v"
if not "!HAVE_GO!"=="%WANT_GO%" (
    echo [gen] protoc-gen-go version mismatch: have "!HAVE_GO!", want "%WANT_GO%"
    echo [gen]   go install google.golang.org/protobuf/cmd/protoc-gen-go@%WANT_GO%
    exit /b 1
)

rem Keep this list in sync with gen.sh; add new .proto dirs to both.
set "PROTO_NAMES=auth call conversation egress errinfo group jssdk livekit_meeting meeting_room msg msggateway oa push relation rtc schedule sdkws statistics third user wrapperspb"

for %%i in (%PROTO_NAMES%) do (
    protoc --go_out=./%%i --go_opt=module=github.com/openimsdk/protocol/%%i %%i/%%i.proto
    if ERRORLEVEL 1 (
        echo error processing %%i.proto ^(go_out^)
        exit /b %ERRORLEVEL%
    )
)

rem Generate Go-grpc code

for %%i in (%PROTO_NAMES%) do (
    protoc --go-grpc_out=./%%i --go-grpc_opt=module=github.com/openimsdk/protocol/%%i %%i/%%i.proto
    if ERRORLEVEL 1 (
        echo error processing %%i.proto ^(go-grpc_out^)
        exit /b %ERRORLEVEL%
    )
)

rem Strip omitempty from json tags (this fork serializes zero values).
rem Every ",omitempty" in generated code is the tail of a json tag value, so an
rem unanchored strip is safe -- and it is required: map fields render as
rem   json:"maxSeqs,omitempty" protobuf_key:"..." protobuf_val:"..."
rem so a backtick-anchored pattern would miss all 26 of them.
rem
rem Repo-wide by design: this also covers openmeeting/*, which gen_openmeeting.sh
rem generates. The fork convention is "no omitempty anywhere", not "no omitempty
rem in what this script regenerates".
rem
rem Write back via .NET WriteAllText + UTF8Encoding($false): Windows PowerShell
rem 5.1's `Set-Content -Encoding UTF8` writes UTF-8 *with* BOM, which prepends
rem EF BB BF to every file and makes the whole repo show up as modified.
for /r %%f in (*.pb.go) do (
    powershell -NoProfile -Command "$p='%%f'; $t=[IO.File]::ReadAllText($p); $t=$t.Replace(',omitempty',''); [IO.File]::WriteAllText($p,$t,(New-Object Text.UTF8Encoding $false))"
    if ERRORLEVEL 1 (
        echo error stripping omitempty in %%f
        exit /b %ERRORLEVEL%
    )
)

endlocal
