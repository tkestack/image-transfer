// +build !containers_image_storage_stub

package alltransports

import (
	// Register the storage transport
	_ "tkestack.io/image-transfer/pkg/container-image/storage"
)
