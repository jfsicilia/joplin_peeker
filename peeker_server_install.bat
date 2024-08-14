REM Description: Install peeker_server in c:\tools\peeker_server
@echo off
cls
echo "Building peeker_server.exe"
go build -o peeker_server.exe peeker_server.go
echo "Installing peeker_server.exe in c:\tools\peeker_server"
mkdir c:\tools\peeker_server
copy peeker_server.exe c:\tools\peeker_server
mkdir c:\tools\peeker_server\static
xcopy static c:\tools\peeker_server\static /E /Y /C /I /H
