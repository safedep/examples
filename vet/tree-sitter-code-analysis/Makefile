BIN_DIR := bin
BIN_CALLGRAPH := $(BIN_DIR)/cg

all: create_bin callgraph

.PHONY: create_bin
create_bin:
	mkdir -p $(BIN_DIR)

.PHONY: callgraph
callgraph:
	go build -o $(BIN_CALLGRAPH) cmd/callgraph/main.go

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
