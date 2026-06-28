BINARY   := agq-daemon
INSTALL  := /usr/local/bin/$(BINARY)
UNIT_DIR := $(HOME)/.config/systemd/user
UNIT     := agq.service
CMD      := ./cmd/agq-daemon
BUILDFLAGS ?= -buildvcs=false

.PHONY: build test install enable disable uninstall clean run

## build — compile the daemon binary
build:
	go build $(BUILDFLAGS) -o $(BINARY) $(CMD)

## test — run all package tests
test:
	go test ./...

## run — build and run locally (Ctrl-C to stop)
run: build
	./$(BINARY)

## install — copy binary + unit file, reload systemd
install: build
	sudo cp $(BINARY) $(INSTALL)
	mkdir -p $(UNIT_DIR)
	cp $(UNIT) $(UNIT_DIR)/$(UNIT)
	systemctl --user daemon-reload
	@echo "Installed. Run 'make enable' to start on login."

## enable — enable and start the service now
enable:
	systemctl --user enable --now $(UNIT)
	@echo "Service enabled. Check status with: systemctl --user status agq"

## disable — stop and disable the service
disable:
	systemctl --user disable --now $(UNIT)

## uninstall — remove binary and unit file
uninstall: disable
	sudo rm -f $(INSTALL)
	rm -f $(UNIT_DIR)/$(UNIT)
	systemctl --user daemon-reload

## clean — remove local build artifact
clean:
	rm -f $(BINARY)

## logs — tail the daemon log
logs:
	tail -f $(HOME)/.agq/agq.log

## status — show systemd service status
status:
	systemctl --user status $(UNIT)
