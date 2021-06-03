VERSION ?= $(shell git describe --match 'v[0-9]*' --dirty='.m' --tags --always)
REVISION ?= $(shell git rev-parse HEAD)$(shell if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi)

.PHONY: binary clean

binary: 
	@echo "build image-transfer binary"
	@go build -o ./_output/image-transfer ./cmd/image-transfer/main.go

transfer_image: binary
	@echo "build image-transfer image"
	@go build -o ./_output/run hack/containerized/run.go
	@docker build -t ccr.ccs.tencentyun.com/tcrimages/image-transfer:${VERSION} -f hack/containerized/Dockerfile .

clean:
	@rm ./_output/*

