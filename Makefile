NAME=mqtt-stats
BIN=$(NAME)

all: build

clean:
	rm -f $(BIN)

build:
	./build.sh

.PHONY: update-deps clean

