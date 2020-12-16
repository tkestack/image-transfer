// +build !containers_image_docker_daemon_stub

package alltransports

import (
	// Register the docker-daemon transport
	_ "tkestack.io/image-transfer/pkg/container-image/docker/daemon"
)
