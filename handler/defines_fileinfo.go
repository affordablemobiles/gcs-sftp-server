package gsftp

import (
	"os"
	"time"

	"cloud.google.com/go/storage"
)

type SyntheticFileInfo struct {
	objAttr *storage.ObjectAttrs
	prefix  string
}

func (f *SyntheticFileInfo) Name() string { // base name of the file
	if f.objAttr.Prefix != "" {
		return f.objAttr.Prefix[len(f.prefix) : len(f.objAttr.Prefix)-1]
	}

	return f.objAttr.Name[len(f.prefix):]
}

func (f *SyntheticFileInfo) Size() int64 { // length in bytes for regular files; system-dependent for others
	return f.objAttr.Size
}

func (f *SyntheticFileInfo) Mode() os.FileMode { // file mode bits
	if f.IsDir() {
		return os.ModeDir | 0777
	}

	return 0777
}

func (f *SyntheticFileInfo) ModTime() time.Time { // modification time
	if f.IsDir() {
		return time.Now()
	}

	return f.objAttr.Updated
}

func (f *SyntheticFileInfo) IsDir() bool { // abbreviation for Mode().IsDir()
	if f.objAttr.Prefix != "" {
		return true
	} else if len(f.objAttr.Name) > 0 {
		if f.objAttr.Name[len(f.objAttr.Name)-1:] == "/" && f.objAttr.Size == 0 {
			return true
		}
	}

	return false
}

func (f *SyntheticFileInfo) Sys() interface{} { // underlying data source (can return nil)
	return nil
}
