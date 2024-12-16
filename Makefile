BINARY_NAME=elcentral

.PHONY: build

build:
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(BINARY_NAME)_rpi ./cmd/metrics_collector
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)_mac ./cmd/metrics_collector

clean:
	rm -f $(BINARY_NAME)_rpi $(BINARY_NAME)_mac
