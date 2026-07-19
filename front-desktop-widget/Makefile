# Makefile for Todo Planner Desktop Widget
# macOS Swift application using swiftc

APP_NAME = TodoPlannerWidget
BUNDLE_ID = com.todoplanner.widget
BUILD_DIR = build
APP_BUNDLE = $(BUILD_DIR)/$(APP_NAME).app
CONTENTS = $(APP_BUNDLE)/Contents
MACOS_DIR = $(CONTENTS)/MacOS
RESOURCES = $(CONTENTS)/Resources
APP_EXECUTABLE = $(MACOS_DIR)/$(APP_NAME)

SOURCES = $(shell find Sources -name '*.swift' -type f | LC_ALL=C sort)
ARCH ?= $(shell uname -m)

# Detect Swift compiler (Swiftly or system)
SWIFTC := $(shell command -v swiftc 2> /dev/null)
SWIFT := $(shell command -v swift 2> /dev/null)

ifndef SWIFTC
$(error Swift compiler not found. Install via Swiftly: curl -L https://swift-server.github.io/swiftly/swiftly-install.sh | bash)
endif

.PHONY: all clean run dev help test install

help: ## Show this help message
	@echo "Todo Planner Desktop Widget - macOS Swift App"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

all: $(APP_EXECUTABLE) ## Build the application

$(APP_EXECUTABLE): $(SOURCES) Info.plist
	@mkdir -p "$(MACOS_DIR)" "$(RESOURCES)"
ifeq ($(ARCH),universal)
	@echo "Building universal binary..."
	$(SWIFTC) \
		-parse-as-library \
		-o "$(MACOS_DIR)/$(APP_NAME)-arm64" \
		-sdk $$(xcrun --show-sdk-path) \
		-target arm64-apple-macosx14.0 \
		$(SOURCES)
	$(SWIFTC) \
		-parse-as-library \
		-o "$(MACOS_DIR)/$(APP_NAME)-x86_64" \
		-sdk $$(xcrun --show-sdk-path) \
		-target x86_64-apple-macosx14.0 \
		$(SOURCES)
	lipo -create -output "$(MACOS_DIR)/$(APP_NAME)" \
		"$(MACOS_DIR)/$(APP_NAME)-arm64" \
		"$(MACOS_DIR)/$(APP_NAME)-x86_64"
	@rm "$(MACOS_DIR)/$(APP_NAME)-arm64" "$(MACOS_DIR)/$(APP_NAME)-x86_64"
else
	@echo "Building for $(ARCH)..."
	$(SWIFTC) \
		-parse-as-library \
		-o "$(MACOS_DIR)/$(APP_NAME)" \
		-sdk $$(xcrun --show-sdk-path) \
		-target $(ARCH)-apple-macosx14.0 \
		$(SOURCES)
endif
	@cp Info.plist "$(CONTENTS)/"
	@plutil -replace CFBundleName -string "$(APP_NAME)" "$(CONTENTS)/Info.plist"
	@plutil -replace CFBundleDisplayName -string "$(APP_NAME)" "$(CONTENTS)/Info.plist"
	@plutil -replace CFBundleExecutable -string "$(APP_NAME)" "$(CONTENTS)/Info.plist"
	@plutil -replace CFBundleIdentifier -string "$(BUNDLE_ID)" "$(CONTENTS)/Info.plist"
	@echo "✓ Built $(APP_BUNDLE)"

run: all ## Build and run the application
	@echo "Launching $(APP_NAME)..."
	@open "$(APP_BUNDLE)"

dev: all ## Run in development mode (with console output)
	@echo "Running $(APP_NAME) in development mode..."
	@"$(APP_EXECUTABLE)"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "✓ Clean complete"

install: all ## Install the app to /Applications
	@echo "Installing to /Applications..."
	@cp -r "$(APP_BUNDLE)" /Applications/
	@echo "✓ Installed to /Applications/$(APP_NAME).app"

universal: ## Build universal binary (arm64 + x86_64)
	@$(MAKE) ARCH=universal all

logs: ## Stream application logs
	@echo "Streaming logs from $(BUNDLE_ID)..."
	@echo "Press Ctrl+C to stop"
	@log stream --level info --predicate 'subsystem == "$(BUNDLE_ID)"' --style compact

setup: ## Initial setup (check dependencies)
	@echo "Checking Swift toolchain..."
	@command -v swiftc >/dev/null 2>&1 || { \
		echo "Swift not found. Install via Swiftly:"; \
		echo "  curl -L https://swift-server.github.io/swiftly/swiftly-install.sh | bash"; \
		exit 1; \
	}
	@echo "Swift version: $$($(SWIFTC) --version | head -n1)"
	@echo ""
	@echo "Checking for Xcode Command Line Tools (for SDK)..."
	@xcode-select -p >/dev/null 2>&1 || xcode-select --install
	@echo "✓ Setup complete!"
	@echo ""
	@echo "Optional tools (install manually if needed):"
	@echo "  brew install swiftformat  # for code formatting"
	@echo "  brew install swiftlint    # for code linting"
