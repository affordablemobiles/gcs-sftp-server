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
	if f.objAttr.Prefix == "" {
		return f.objAttr.Name[len(f.prefix):]
	} else {
		return f.objAttr.Prefix[len(f.prefix) : len(f.objAttr.Prefix)-1]
	}
}

func (f *SyntheticFileInfo) Size() int64 { // length in bytes for regular files; system-dependent for others
	return f.objAttr.Size
}

func (f *SyntheticFileInfo) Mode() os.FileMode { // file mode bits
	if f.objAttr.Prefix == "" {
		return 0777
	} else {
		return os.ModeDir | 0777
	}
}

func (f *SyntheticFileInfo) ModTime() time.Time { // modification time
	if f.objAttr.Prefix == "" {
		return f.objAttr.Updated
	} else {
		return time.Now()
	}
}

func (f *SyntheticFileInfo) IsDir() bool { // abbreviation for Mode().IsDir()
	return f.objAttr.Prefix == ""
}

func (f *SyntheticFileInfo) Sys() interface{} { // underlying data source (can return nil)
	return nil
}
