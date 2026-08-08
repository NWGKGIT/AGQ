BINARY   := agq-daemon
INSTALL  := /usr/local/bin/$(BINARY)
UNIT_DIR := $(HOME)/.config/systemd/user
UNIT     := agq.service
CMD      := ./cmd/agq-daemon
BUILDFLAGS ?= -buildvcs=false

.PHONY: build test install enable disable uninstall clean run desktop-build desktop-dev desktop-test desktop-appimage brand-assets docker-build docker-test

## build — compile the daemon binary
build:
	go build $(BUILDFLAGS) -o $(BINARY) $(CMD)

## test — run all package tests
test:
	go test ./...

## docker-build — build the containerized headless daemon image
docker-build:
	docker build -t agq-daemon .

## docker-test — run vet + the Go test suite inside a container
docker-test:
	docker build --target test -t agq-test .

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
	@echo "Service enabled. Check status with: systemctl --user status $(UNIT)"

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

## desktop-build — build the desktop app (requires wails CLI)
desktop-build:
	cd desktop && wails build -tags webkit2_41

## desktop-dev — run the desktop app with hot reload
desktop-dev:
	cd desktop && wails dev -tags webkit2_41

## desktop-test — run desktop Go tests and frontend typecheck/build
desktop-test:
	cd desktop && go test ./...
	cd desktop/frontend && npm test
	cd desktop/frontend && npm run build

## brand-assets — regenerate Wails, Windows Store, and AppImage icons
brand-assets:
	cd desktop && ./packaging/generate-assets.sh

## desktop-appimage — build the x86_64 AppImage release artifact
desktop-appimage:
	cd desktop && ./packaging/linux/build-appimage.sh

## logs — tail the daemon log
logs:
	tail -f $(HOME)/.agq/agq.log

## status — show systemd service status
status:
	systemctl --user status $(UNIT)
