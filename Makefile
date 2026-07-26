VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/Yashh56/atlas/internal/version.Version=$(VERSION)

.PHONY: build clean test install winres

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o atlas$(if $(COMSPEC),.exe,) ./cmd/atlas

test:
	go test ./...

clean:
	go clean -cache
	rm -f atlas atlas.exe

install: build
	go install -trimpath -ldflags="$(LDFLAGS)" ./cmd/atlas

winres:
	go run github.com/tc-hib/go-winres@latest make --in cmd/atlas/winres.json --out cmd/atlas/rsrc
