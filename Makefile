BINARY := agq-daemon
CMD    := ./cmd/agq-daemon

.PHONY: build test run clean

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
