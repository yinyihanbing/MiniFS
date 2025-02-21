chcp 65001
echo off

set run_path=%~dp0%

cd main

go build -o %run_path%/bin/mini_web_dir.exe

set CGO_ENABLED=0
set GOARCH=amd64
set GOOS=linux

go build -o %run_path%/bin/mini_web_dir