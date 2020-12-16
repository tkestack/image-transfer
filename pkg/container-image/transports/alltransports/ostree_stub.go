// +build !containers_image_ostree !linux

package alltransports

import "tkestack.io/image-transfer/pkg/container-image/transports"

func init() {
	transports.Register(transports.NewStubTransport("ostree"))
}
