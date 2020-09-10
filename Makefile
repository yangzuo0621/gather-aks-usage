BINARY_DEST_DIR ?= bin

.PHONY: cli
build-cli:
	go build ${GO_BUILD_OPTIONS} -o ${BINARY_DEST_DIR}/gather-aks-usage github.com/yangzuo0621/gather-aks-usage