package system

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gravitational/robotest/lib/constants"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/trace"
)

// CopyFile copies contents of src to dst atomically
// using SharedReadWriteMask as permissions.
func CopyFile(dst, src string) error {
	return CopyFileWithPerms(dst, src, constants.SharedReadWriteMask)
}

// CopyFileWithPerms copies the contents from src to dst atomically.
// If dst does not exist, CopyFile creates it with permissions perm.
// If the copy fails, CopyFile aborts and dst is preserved.
// Adopted with modifications from https://go-review.googlesource.com/#/c/1591/9/src/io/ioutil/ioutil.go
func CopyFileWithPerms(dst, src string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	defer in.Close()
	tmp, err := ioutil.TempFile(filepath.Dir(dst), "")
	if err != nil {
		return trace.ConvertSystemError(err)
	}

	cleanup := func() error {
		err := os.Remove(tmp.Name())
		if err != nil {
			log.Errorf("failed to remove %q: %v", tmp.Name(), err)
		}
		return trace.ConvertSystemError(err)
	}

	_, err = io.Copy(tmp, in)
	if err != nil {
		tmp.Close()
		cleanup()
		return trace.ConvertSystemError(err)
	}
	if err = tmp.Close(); err != nil {
		cleanup()
		return trace.ConvertSystemError(err)
	}
	if err = os.Chmod(tmp.Name(), perm); err != nil {
		cleanup()
		return trace.ConvertSystemError(err)
	}
	err = os.Rename(tmp.Name(), dst)
	if err != nil {
		cleanup()
	}
	return nil
}

// RemoveALl removes the specified directory including sub-directories
func RemoveAll(dir string) error {
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
	return os.Remove(dir)
}
