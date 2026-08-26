BINARY   := agq-daemon
INSTALL  := /usr/local/bin/$(BINARY)
UNIT_DIR := $(HOME)/.config/systemd/user
UNIT     := agq.service
CMD      := ./cmd/agq-daemon
BUILDFLAGS ?= -buildvcs=false

# Release version.
#
# Automatically uses the latest release tag such as v1.1.0 -> 1.1.0.
# Override explicitly with:
#   make VERSION=1.2.0 desktop-rpm
VERSION ?= $(shell git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null | sed 's/^v//')

.PHONY: build test install enable disable uninstall clean run logs status \
		desktop-build desktop-dev desktop-test \
        desktop-deb desktop-rpm desktop-arch desktop-linux-packages \
        brand-assets docker-build docker-test check-version

## build - compile the daemon binary
build:
	go build $(BUILDFLAGS) -o $(BINARY) $(CMD)

## test - run all package tests
test:
	go test ./...

## docker-build - build the containerized headless daemon image
docker-build:
	docker build -t agq-daemon .

## docker-test - run vet + the Go test suite inside a container
docker-test:
	docker build --target test -t agq-test .

## run - build and run locally (Ctrl-C to stop)
run: build
	./$(BINARY)

## install - copy binary + unit file, reload systemd
install: build
	sudo cp $(BINARY) $(INSTALL)
	mkdir -p $(UNIT_DIR)
	cp $(UNIT) $(UNIT_DIR)/$(UNIT)
	systemctl --user daemon-reload
	@echo "Installed. Run 'make enable' to start on login."

## enable - enable and start the service now
enable:
	systemctl --user enable --now $(UNIT)
	@echo "Service enabled. Check status with: systemctl --user status $(UNIT)"

## disable - stop and disable the service
disable:
	systemctl --user disable --now $(UNIT)

## uninstall - remove binary and unit file
uninstall:
	-systemctl --user disable --now $(UNIT)
	sudo rm -f $(INSTALL)
	rm -f $(UNIT_DIR)/$(UNIT)
	systemctl --user daemon-reload

## clean - remove local build artifact
clean:
	rm -f $(BINARY)

## desktop-build - build the desktop app
desktop-build:
	cd desktop && wails build -tags webkit2_41

## desktop-dev - run the desktop app with hot reload
desktop-dev:
	cd desktop && wails dev -tags webkit2_41

## desktop-test - run desktop Go tests and frontend typecheck/build
desktop-test:
	cd desktop/frontend && npm run build
	cd desktop && go test ./...
	cd desktop/frontend && npm test

## brand-assets - regenerate Wails and Windows Store icons
brand-assets:
	cd desktop && ./packaging/generate-assets.sh

## check-version - ensure a valid package version exists
check-version:
	@test -n "$(VERSION)" || { \
		echo "ERROR: No release version found."; \
		echo "Use VERSION=1.1.0 or create a release tag such as v1.1.0."; \
		exit 1; \
	}

## desktop-deb - build the Debian package
desktop-deb: check-version
	cd desktop && VERSION=$(VERSION) ./packaging/linux/build-native-packages.sh deb

## desktop-rpm - build the RPM package
desktop-rpm: check-version
	cd desktop && VERSION=$(VERSION) ./packaging/linux/build-native-packages.sh rpm

## desktop-arch - build the Arch package
desktop-arch: check-version
	cd desktop && VERSION=$(VERSION) ./packaging/linux/build-native-packages.sh arch

## desktop-linux-packages - build all native Linux packages
desktop-linux-packages: check-version
	cd desktop && VERSION=$(VERSION) ./packaging/linux/build-native-packages.sh

## logs - tail the daemon log
logs:
	tail -f $(HOME)/.agq/agq.log

## status - show systemd service status
status:
	systemctl --user status $(UNIT)
