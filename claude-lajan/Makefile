BINARY      := lajan
BIN_DIR     := $(HOME)/.claude-lajan/bin
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build install uninstall clean test

## build: compile all binaries locally (for testing)
build:
	go build -o $(BINARY)       ./cmd/lajan
	go build -o stop-hook       ./cmd/hook/stop
	go build -o prompt-hook     ./cmd/hook/prompt

## install: build all binaries, deploy them, register hooks in settings.json
install:
	@mkdir -p $(BIN_DIR) $(INSTALL_DIR)
	@echo "→ Building binaries..."
	go build -o $(BIN_DIR)/lajan       ./cmd/lajan
	go build -o $(BIN_DIR)/stop-hook    ./cmd/hook/stop
	go build -o $(BIN_DIR)/prompt-hook  ./cmd/hook/prompt
	go build -o $(BIN_DIR)/pretool-hook ./cmd/hook/pretool
	@echo "→ Linking lajan to $(INSTALL_DIR)/lajan"
	@ln -sf $(BIN_DIR)/lajan $(INSTALL_DIR)/lajan
	@echo "→ Registering hooks in ~/.claude/settings.json"
	$(BIN_DIR)/lajan install
	@echo ""
	@echo "Done. Start a new Claude Code session to activate."
	@echo "After each session, analysis runs automatically in the background."
	@echo "Output: ~/.claude-lajan/lajan.log"

## uninstall: remove hooks from settings.json, delete binaries and symlink
uninstall:
	@if [ -x $(BIN_DIR)/lajan ]; then \
		echo "→ Removing hooks from settings.json..."; \
		$(BIN_DIR)/lajan reset --hooks --yes; \
	fi
	@rm -f $(BIN_DIR)/lajan $(BIN_DIR)/stop-hook $(BIN_DIR)/prompt-hook $(BIN_DIR)/pretool-hook
	@rm -f $(INSTALL_DIR)/lajan
	@echo "→ Uninstalled."

## clean: remove local build artefacts
clean:
	rm -f $(BINARY) stop-hook prompt-hook

## test: run all tests
test:
	go test ./...
