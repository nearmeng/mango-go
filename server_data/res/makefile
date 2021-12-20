PROTOC_PATH=../../tools/protoc/protoc
PLUGIN_GO_PATH=../../tools/protoc/protoc-gen-go

.PHONY: proto

proto:
	${PROTOC_PATH} xres/xres_header.proto --go_out=. --plugin=protoc-gen-go=${PLUGIN_GO_PATH}