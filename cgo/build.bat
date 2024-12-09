go mod tidy
go build -buildmode=c-shared -ldflags="-s -w" -o dll\microSdk.dll main.go