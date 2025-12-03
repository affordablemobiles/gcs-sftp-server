package gsftp

import (
	"io"
	"os"
	"sync"
)

// --- Memory Buffer (Default) ---

type WriteAtBuffer struct {
	buf []byte
	m   sync.Mutex

	// GrowthCoeff defines the growth rate of the internal buffer. By
	// default, the growth rate is 1, where expanding the internal
	// buffer will allocate only enough capacity to fit the new expected
	// length.
	GrowthCoeff float64

	Writer io.WriteCloser
}

// NewWriteAtBuffer creates a WriteAtBuffer with an internal buffer
// provided by buf.
func NewWriteAtBuffer(w io.WriteCloser, buf []byte) *WriteAtBuffer {
	return &WriteAtBuffer{
		Writer: w,
		buf:    buf,
	}
}

// WriteAt writes a slice of bytes to a buffer starting at the position provided
// The number of bytes written will be returned, or error. Can overwrite previous
// written slices if the write ats overlap.
func (b *WriteAtBuffer) WriteAt(p []byte, pos int64) (n int, err error) {
	pLen := len(p)
	expLen := pos + int64(pLen)
	b.m.Lock()
	defer b.m.Unlock()
	if int64(len(b.buf)) < expLen {
		if int64(cap(b.buf)) < expLen {
			if b.GrowthCoeff < 1 {
				b.GrowthCoeff = 1
			}
			newBuf := make([]byte, expLen, int64(b.GrowthCoeff*float64(expLen)))
			copy(newBuf, b.buf)
			b.buf = newBuf
		}
		b.buf = b.buf[:expLen]
	}
	copy(b.buf[pos:], p)
	return pLen, nil
}

// Bytes returns a slice of bytes written to the buffer.
func (b *WriteAtBuffer) Bytes() []byte {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buf
}

func (b *WriteAtBuffer) Close() error {
	b.m.Lock()
	defer b.m.Unlock()

	_, err := b.Writer.Write(b.buf)
	if err != nil {
		return err
	}

	return b.Writer.Close()
}

// --- Disk Buffer (Optional) ---

type DiskWriteAtBuffer struct {
	file   *os.File
	path   string
	Writer io.WriteCloser
}

func NewDiskWriteAtBuffer(w io.WriteCloser, tempDir string) (*DiskWriteAtBuffer, error) {
	f, err := os.CreateTemp(tempDir, "sftp-upload-*")
	if err != nil {
		return nil, err
	}

	return &DiskWriteAtBuffer{
		file:   f,
		path:   f.Name(),
		Writer: w,
	}, nil
}

func (b *DiskWriteAtBuffer) WriteAt(p []byte, pos int64) (n int, err error) {
	return b.file.WriteAt(p, pos)
}

func (b *DiskWriteAtBuffer) Close() error {
	defer os.Remove(b.path) // Ensure temp file is cleaned up

	// Ensure all writes are flushed to disk
	if err := b.file.Sync(); err != nil {
		b.file.Close()
		return err
	}

	// Rewind to the beginning
	if _, err := b.file.Seek(0, 0); err != nil {
		b.file.Close()
		return err
	}

	// Stream from disk to GCS
	if _, err := io.Copy(b.Writer, b.file); err != nil {
		b.file.Close()
		return err
	}

	// Close the local file
	if err := b.file.Close(); err != nil {
		return err
	}

	// Close the GCS writer to finalize the upload
	return b.Writer.Close()
}
