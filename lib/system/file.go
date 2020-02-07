package system

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gravitational/robotest/lib/constants"

	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

// CopyFile copies contents of src to dst atomically
// using SharedReadWriteMask as permissions.
func CopyFile(src, dst string) error {
	log.Debugf("copy %s -> %s", src, dst)
	return CopyFileWithPerms(src, dst, constants.SharedReadWriteMask)
}

// CopyFileWithPerms copies the contents from src to dst atomically.
// If dst does not exist, CopyFile creates it with permissions perm.
// If the copy fails, CopyFile aborts and dst is preserved.
// Adopted with modifications from https://go-review.googlesource.com/#/c/1591/9/src/io/ioutil/ioutil.go
func CopyFileWithPerms(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	defer in.Close()
	tmp, err := ioutil.TempFile(filepath.Dir(dst), "")
	if err != nil {
		return trace.ConvertSystemError(err)
	}

	cleanup := func() {
		err := os.Remove(tmp.Name())
		if err != nil {
			log.Warnf("Failed to remove %v: %v.", tmp.Name(), err)
		}
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
		return trace.ConvertSystemError(err)
	}
	return nil
}

// Recursively copy
// dst must always be a directory
// src may be either a dir or a file
func CopyAll(src, dst string) (fileCount uint, err error) {
	fileCount = 0
	err = copyAll(src, dst, &fileCount)
	return fileCount, err
}

func copyAll(src, dst string, fileCount *uint) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	log.Debugf("copy %s -> %s", src, dst)

	// fetch source permissions
	si, err := os.Stat(src)
	if err != nil {
		return trace.ConvertSystemError(err)
	}

	// files often don't have execute, but directories need it
	mode := si.Mode()
	if si.Mode().IsRegular() {
		mode = mode | os.ModeDir
		mode = mode | 0100
	}

	// ensure destination exists
	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return trace.ConvertSystemError(err)
	}

	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(dst, mode)
		if err != nil {
			return trace.ConvertSystemError(err)
		}
	}

	// if source is a file, copy
	if si.Mode().IsRegular() {
		*fileCount++
		return CopyFile(src, filepath.Join(dst, filepath.Base(src)))
	}

	// if sorce is a directory, copy all contents
	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return trace.ConvertSystemError(err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyAll(srcPath, dstPath, fileCount)
			if err != nil {
				return trace.Wrap(err)
			}
			continue
		}

		err = CopyFile(srcPath, dstPath)
		if err != nil {
			return trace.Wrap(err)
		}
		*fileCount++
	}

	return nil
}

// RemoveAll removes the specified directory including sub-directories
func RemoveAll(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return trace.ConvertSystemError(err)
		}
	}
	return trace.ConvertSystemError(os.Remove(dir))
}
