export GOOS="linux"
go build -o build/dcns-linux

export GOOS="windows"
go build -o build/dcns-windows.exe