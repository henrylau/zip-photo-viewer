CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" GOOS=windows GOARCH=amd64 go build



CGO_ENABLED=1 CGO_CFLAGS_ALLOW="-Xpreprocessor" CGO_CFLAGS="-I/usr/local/libexec/gcc/x86_64-w64-mingw32/12.2.0/include" CC="x86_64-w64-mingw32-gcc" GOOS=windows GOARCH=amd64 go build
CGO_ENABLED=1 CGO_CFLAGS_ALLOW="-Xpreprocessor" CC="x86_64-w64-mingw32-gcc" GOOS=windows GOARCH=amd64 go build

CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build

