@echo off
cd /d C:\Users\eduar\Downloads\picoclaw_Windows_x86_64
start "" http://127.0.0.1:18800
bin\picoclaw-dev.exe web -config run\config.json -addr 127.0.0.1:18800
