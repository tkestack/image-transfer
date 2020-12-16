// +build containers_image_storage_stub

package alltransports

import "tkestack.io/image-transfer/pkg/container-image/transports"

func init() {
	transports.Register(transports.NewStubTransport("containers-storage"))
}
