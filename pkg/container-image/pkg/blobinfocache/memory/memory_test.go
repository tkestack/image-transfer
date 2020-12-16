package memory

import (
	"testing"

	"tkestack.io/image-transfer/pkg/container-image/pkg/blobinfocache/internal/test"
	"tkestack.io/image-transfer/pkg/container-image/types"
)

func newTestCache(t *testing.T) (types.BlobInfoCache, func(t *testing.T)) {
	return New(), func(t *testing.T) {}
}

func TestNew(t *testing.T) {
	test.GenericCache(t, newTestCache)
}
