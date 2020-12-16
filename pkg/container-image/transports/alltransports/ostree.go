// +build containers_image_ostree,linux

package alltransports

import (
	// Register the ostree transport
	_ "tkestack.io/image-transfer/pkg/container-image/ostree"
)
