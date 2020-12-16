.PHONY: cmd clean

cmd: $(wildcard ./pkg/flag/*.go ./pkg/container-image/*.go ./pkg/log/*.go ./pkg/image-transfer/*.go ./pkg/utils/*.go ./cmd/image-transfer/*.go ./configs/*.go ./*.go)
	go build -o image-transfer ./cmd/image-transfer/main.go


clean:
	rm image-transfer