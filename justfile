version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

web:
    cd web && npm run build
    rm -rf static && cp -r web/build static

build: web
    go build -ldflags "-X main.version={{version}}" -o bin/axon-book ./example/

install: build
    cp bin/axon-book ${AURELIA_ROOT}/bin/book
    @echo "Installed book {{version}}"

test:
    go vet ./...
    go test ./...
