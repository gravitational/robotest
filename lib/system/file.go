package system

import (
	"io"
	"os"
	"path/filepath"

	"github.com/gravitational/trace"
)

func CopyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return trace.Wrap(err)
	}
	defer in.Close()

	out, err := os.Create(to)
	if err != nil {
		return trace.Wrap(err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// RemoveContents removes the contents of the specified directory including any
// sub-directories
func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return trace.Wrap(err)
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return trace.Wrap(err)
		}
	}
	return nil
}
