go build -ldflags "-w -s" -o bin/client.exe .\client\client.go

go build -ldflags "-w -s" -o bin/server.exe .\server\server.go

upx bin/client.exe
upx bin/server.exe

echo "Build complete."
