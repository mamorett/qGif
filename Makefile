APP_ID  ?= io.github.qgif.qgif
SDK     ?=

.PHONY: all run test build package cross-linux cross-windows cross-darwin

all: build

build:
	go build -trimpath -ldflags="-s -w" -o bin/qgif .

run: build
	./bin/qgif

test:
	go test ./internal/...

package:
	fyne package

cross-linux:
	fyne-cross linux -arch=amd64,arm64 -app-id $(APP_ID)

cross-windows:
	fyne-cross windows -arch=amd64,arm64 -app-id $(APP_ID)

cross-darwin:
	fyne-cross darwin -arch=amd64,arm64 -app-id $(APP_ID) -macosx-sdk-path $(SDK)
