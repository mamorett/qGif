# ─────────────────────────────────────────────────────────────────────────────
# qGif — Build Configuration
# ─────────────────────────────────────────────────────────────────────────────

APP_NAME    := qGif
APP_ID      := qgif
BINARY_NAME := qgif
MODULE      := github.com/mamorett/qgif

# Paths
BUILD_DIR   := build
ICON_NAME   := Icon.png
ICON_SRC    := $(ICON_NAME)

# Tools
FYNE        := $(HOME)/go/bin/fyne

# Install paths (Linux)
INSTALL_BIN  := $(HOME)/bin
INSTALL_APPS := $(HOME)/.local/share/applications
INSTALL_ICON := $(HOME)/.local/share/icons/hicolor

# ─────────────────────────────────────────────────────────────────────────────
# Detect host OS
# ─────────────────────────────────────────────────────────────────────────────

OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')

# ─────────────────────────────────────────────────────────────────────────────
# Phony targets
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: all build install install-linux clean test

# ─────────────────────────────────────────────────────────────────────────────
# Default target
# ─────────────────────────────────────────────────────────────────────────────

all: build

# ─────────────────────────────────────────────────────────────────────────────
# Tests
# ─────────────────────────────────────────────────────────────────────────────

test:
	go test ./...

# ─────────────────────────────────────────────────────────────────────────────
# Build (auto-detects host OS)
# ─────────────────────────────────────────────────────────────────────────────

build:
	mkdir -p $(BUILD_DIR)
ifeq ($(OS),darwin)
	@echo "→ Building macOS app bundle…"
	CGO_ENABLED=1 $(FYNE) package \
		-os darwin -name $(APP_NAME) -icon $(ICON_NAME) -id $(APP_ID)
	mv $(APP_NAME).app $(BUILD_DIR)/
else ifeq ($(OS),linux)
	@echo "→ Building Linux binary…"
	GOOS=linux CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) .
else
	$(error Unsupported OS: $(OS))
endif

# ─────────────────────────────────────────────────────────────────────────────
# Install
# ─────────────────────────────────────────────────────────────────────────────

install: install-linux

install-linux: build
ifneq ($(OS),linux)
	$(error install-linux is only supported on Linux)
endif
	@echo "→ Removing stale desktop entries…"
	rm -f $(INSTALL_APPS)/$(APP_ID).desktop
	rm -f $(INSTALL_APPS)/io.github.mamorett.$(APP_ID).desktop

	@echo "→ Installing binaries…"
	install -Dm755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_BIN)/$(BINARY_NAME)

	@echo "→ Installing icons…"
	install -Dm644 $(ICON_SRC) $(INSTALL_ICON)/256x256/apps/$(APP_ID).png
	install -Dm644 $(ICON_SRC) $(INSTALL_ICON)/128x128/apps/$(APP_ID).png
	install -Dm644 $(ICON_SRC) $(INSTALL_ICON)/48x48/apps/$(APP_ID).png

	@echo "→ Installing desktop entry…"
	mkdir -p $(INSTALL_APPS)
	cp assets/linux/$(APP_ID).desktop $(INSTALL_APPS)/$(APP_ID).desktop
	sed -i "s|^Exec=.*|Exec=$(INSTALL_BIN)/$(BINARY_NAME)|" \
		$(INSTALL_APPS)/$(APP_ID).desktop

	@echo "→ Refreshing desktop and icon caches…"
	update-desktop-database $(INSTALL_APPS)/ 2>/dev/null || true
	gtk-update-icon-cache -f -t $(INSTALL_ICON)/ 2>/dev/null || true
	kbuildsycoca6 2>/dev/null || kbuildsycoca5 2>/dev/null || true

	@echo "✓ Installation complete."

# ─────────────────────────────────────────────────────────────────────────────
# Clean
# ─────────────────────────────────────────────────────────────────────────────

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(APP_NAME).app
	rm -f  $(APP_NAME).tar.xz
