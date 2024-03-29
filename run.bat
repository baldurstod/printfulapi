cls
del .\dist\printfulapi.exe
go build -ldflags="-X printfulapi/src/server.ReleaseMode=false" -o dist/printfulapi.exe ./src/main.go
.\dist\printfulapi.exe
