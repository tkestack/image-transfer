// +build containers_image_docker_daemon_stub

package alltransports

import "tkestack.io/image-transfer/pkg/container-image/transports"

func init() {
	transports.Register(transports.NewStubTransport("docker-daemon"))
}
