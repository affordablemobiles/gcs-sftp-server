package gsftp

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/pkg/sftp"
	"google.golang.org/api/iterator"
)

// Example Handlers
func (fs *gcsHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	object := fs.bucket.Object(r.Filepath[1:])

	log.Printf("Reading file %s", r.Filepath)

	reader, err := object.NewReader(r.Context())
	if err != nil {
		return nil, err
	}

	return NewReadAtBuffer(reader)
}

func (fs *gcsHandler) createDirectoryTree(r *sftp.Request) error {
	dirTree := getDirectoryTree(r.Filepath[1:])

	for _, dir := range dirTree {
		dir = dir + "/"

		object := fs.bucket.Object(dir)

		log.Printf("Checking directory exists for write: /%s", dir)

		_, err := object.Attrs(r.Context())

		if err == storage.ErrObjectNotExist {
			log.Printf("Creating directory for write: /%s", dir)

			writer := object.NewWriter(r.Context())

			err := writer.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (fs *gcsHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	err := fs.createDirectoryTree(r)
	if err != nil {
		log.Printf("Failed to create directory tree: %s", err)
		return nil, err
	}

	object := fs.bucket.Object(r.Filepath[1:])

	log.Printf("Writing file %s", r.Filepath)

	writer := object.NewWriter(r.Context())

	return NewWriteAtBuffer(writer, []byte{}), nil
}

func (fs *gcsHandler) Filecmd(r *sftp.Request) error {
	switch r.Method {
	case "Setstat":
		return nil
	case "Rename":
		return fmt.Errorf("not implemented")
	case "Rmdir", "Remove":
		return fmt.Errorf("not implemented")
	case "Mkdir":
		object := fs.bucket.Object(r.Filepath[1:] + "/")

		log.Printf("Creating directory %s", r.Filepath)

		writer := object.NewWriter(r.Context())

		err := writer.Close()
		return err
	case "Symlink":
		return fmt.Errorf("not implemented")
	}
	return nil
}

func (fs *gcsHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	switch r.Method {
	case "List":
		log.Printf("Listing directory for path %s", r.Filepath)

		prefix := r.Filepath[1:]
		if prefix != "" {
			prefix += "/"
		}

		objects := fs.bucket.Objects(r.Context(), &storage.Query{
			Delimiter: "/",
			Prefix:    prefix,
		})

		list := []os.FileInfo{}

		for {
			objAttrs, err := objects.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Error iterating directory %s: %s", r.Filepath, err)

				return nil, err
			}

			// Don't include self.
			if ((prefix != "") && (objAttrs.Prefix == prefix)) || ((objAttrs.Prefix == "") && (objAttrs.Name == prefix)) {
				continue
			}

			list = append(list, &SyntheticFileInfo{
				prefix:  prefix,
				objAttr: objAttrs,
			})
		}

		return listerat(list), nil
	case "Stat":
		if r.Filepath == "/" {
			return listerat([]os.FileInfo{
				&SyntheticFileInfo{
					objAttr: &storage.ObjectAttrs{
						Prefix: "/",
					},
				},
			}), nil
		}

		object := fs.bucket.Object(r.Filepath[1:])

		log.Printf("Getting file info for %s", r.Filepath)

		attrs, err := object.Attrs(r.Context())
		if err == storage.ErrObjectNotExist {
			object := fs.bucket.Object(r.Filepath[1:] + "/")

			log.Printf("Retrying file info for %s", r.Filepath+"/")

			attrs, err = object.Attrs(r.Context())
		}
		if err != nil {
			log.Printf("Failed to get file info for %s: %s", r.Filepath, err)
			if err == storage.ErrObjectNotExist {
				err = os.ErrNotExist
			}
			return nil, err
		}

		file := &SyntheticFileInfo{
			objAttr: attrs,
		}
		return listerat([]os.FileInfo{file}), nil
	case "Readlink":
		return nil, fmt.Errorf("not implemented")
	}
	return nil, nil
}

func getDirectoryTree(dir string) []string {
	dirsToCreate := []string{}

	for {
		dir, _ = filepath.Split(dir)
		if dir == "" {
			break
		}
		dir = filepath.Dir(dir)
		dirsToCreate = append([]string{dir}, dirsToCreate...)
	}

	return dirsToCreate
}
