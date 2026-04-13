BINARY      := lajan
BIN_DIR     := $(HOME)/.claude-lajan/bin
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build install uninstall clean test

## build: compile all binaries locally (for testing)
build:
	go build -o $(BINARY)    ./cmd/lajan
	go build -o stop-hook    ./cmd/hook/stop

## install: build all binaries, deploy them, register hooks in settings.json
install:
	@mkdir -p $(BIN_DIR) $(INSTALL_DIR)
	@echo "→ Building binaries..."
	go build -o $(BIN_DIR)/lajan       ./cmd/lajan
	go build -o $(BIN_DIR)/stop-hook    ./cmd/hook/stop
	@echo "→ Linking lajan to $(INSTALL_DIR)/lajan"
	@ln -sf $(BIN_DIR)/lajan $(INSTALL_DIR)/lajan
	@echo "→ Registering hooks in ~/.claude/settings.json"
	$(BIN_DIR)/lajan install
	@echo ""
	@echo "Done. Start a new Claude Code session to activate."
	@echo "After each session, analysis runs automatically in the background."
	@echo "Output: ~/.claude-lajan/lajan.log"

## uninstall: remove all lajan traces (hooks, learnings, completion, binaries, data dir)
uninstall:
	@if [ -x $(BIN_DIR)/lajan ]; then \
		echo "→ Removing lajan traces from Claude Code and shell config..."; \
		$(BIN_DIR)/lajan uninstall --yes; \
	fi
	@echo "→ Removing binaries and symlink..."
	@rm -f $(BIN_DIR)/lajan $(BIN_DIR)/stop-hook
	@rm -f $(INSTALL_DIR)/lajan
	@if [ -d "$(HOME)/.claude-lajan" ]; then \
		echo "→ Removing ~/.claude-lajan/ data directory..."; \
		rm -rf "$(HOME)/.claude-lajan"; \
	fi
	@if [ -d "$(HOME)/.claude-reviewer" ]; then \
		echo "→ Removing legacy ~/.claude-reviewer directory..."; \
		rm -rf "$(HOME)/.claude-reviewer"; \
	fi
	@echo "→ Uninstalled."

## clean: remove local build artefacts
clean:
	rm -f $(BINARY) stop-hook

## test: run all tests
test:
	go test ./...
