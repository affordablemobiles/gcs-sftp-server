package gsftp

import (
	"io"
	"os"

	"cloud.google.com/go/storage"
)

type gcsHandler struct {
	client *storage.Client
	bucket *storage.BucketHandle
}

type listerat []os.FileInfo

// Modeled after strings.Reader's ReadAt() implementation
func (f listerat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(f)) {
		return 0, io.EOF
	}
	n = copy(ls, f[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}
