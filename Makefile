APP := onionfs

.PHONY: build

build:
	go build -o $(APP) .