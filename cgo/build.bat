go mod tidy
go build -buildmode=c-shared -ldflags="-s -w" -o dll\MicroSdk.Interop.dll main.go