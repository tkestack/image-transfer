.PHONY: binary clean

binary: 
	go build -o ./_output/image-transfer ./cmd/image-transfer/main.go


clean:
	rm ./_output/image-transfer
