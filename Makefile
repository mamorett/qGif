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

# Icon sizes installed into the hicolor theme (a 512px file in a 48x48 slot is
# not a valid hicolor icon, even though Qt tolerates it).
ICON_SIZES  := 16 24 32 48 64 128 256 512
ICON_PNGS   := $(foreach s,$(ICON_SIZES),$(BUILD_DIR)/icons/$(s)x$(s).png)

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

.PHONY: all build icons install install-linux clean test

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
# Icon variants (one properly resized PNG per hicolor size slot)
# ─────────────────────────────────────────────────────────────────────────────

icons: $(ICON_PNGS)

$(BUILD_DIR)/icons/%.png: $(ICON_SRC)
	@mkdir -p $(dir $@)
	@if command -v convert >/dev/null 2>&1; then \
		convert $(ICON_SRC) -strip -resize $(firstword $(subst x, ,$*)) $@; \
	else \
		echo "→ convert(1) not found, installing $(@F) unscaled"; \
		cp $(ICON_SRC) $@; \
	fi

# ─────────────────────────────────────────────────────────────────────────────
# Install
# ─────────────────────────────────────────────────────────────────────────────

install: install-linux

install-linux: build icons
ifneq ($(OS),linux)
	$(error install-linux is only supported on Linux)
endif
	@echo "→ Removing stale desktop entries…"
	rm -f $(INSTALL_APPS)/$(APP_ID).desktop
	rm -f $(INSTALL_APPS)/io.github.mamorett.$(APP_ID).desktop

	@echo "→ Installing binaries…"
	install -Dm755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_BIN)/$(BINARY_NAME)

	@echo "→ Installing icons…"
	@for s in $(ICON_SIZES); do \
		install -Dm644 $(BUILD_DIR)/icons/$${s}x$${s}.png \
			$(INSTALL_ICON)/$${s}x$${s}/apps/$(APP_ID).png; \
	done

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
