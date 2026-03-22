version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

build:
    go build -ldflags "-X main.version={{version}}" -o bin/axon-book ./example/

install: build
    cp bin/axon-book ~/.local/bin/axon-book
    @echo "Installed axon-book {{version}}"

test:
    go vet ./...
    go test ./...
