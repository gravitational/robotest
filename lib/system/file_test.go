package system

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// populates a test workspace (tmp) with a src containing several files
func createSourceDir(t *testing.T) (tmp, src string, fcount int, err error) {
	// create a workspace
	tmp, err = ioutil.TempDir("", "robotest-test")
	if err != nil {
		t.Skipf("unable to create tempdir: %s", err)
	}

	// create content to move, several files to exercise the "all" portion
	src = filepath.Join(tmp, "/src")
	err = os.MkdirAll(src, 0750)
	if err != nil {
		t.Skipf("unable to create src dir: %s", err)
	}

	fcount = 3
	for i := 0; i < fcount; i++ {
		tmpfile, err := ioutil.TempFile(src, "file")
		if err != nil {
			t.Skipf("unable to create tempfile: %s", err)
		}
		defer tmpfile.Close()
		content := []byte("file" + string(i))
		if _, err := tmpfile.Write(content); err != nil {
			t.Skipf("unable to write to %s: %s", tmpfile.Name(), err)
		}
	}
	return tmp, src, fcount, nil
}

func TestCopyAllDirDstPresent(t *testing.T) {
	tmp, src, fcount, err := createSourceDir(t)
	defer os.RemoveAll(tmp)

	dst := filepath.Join(tmp, "/dst")
	err = os.MkdirAll(dst, 0750)
	if err != nil {
		t.Skipf("unable to create dst dir: %s", err)
	}

	var cnt uint
	err = copyAll(src, dst, &cnt)
	if err != nil {
		t.Errorf("copy %q -> %q failed: %s", src, dst, err)
	}
	if int(cnt) != fcount {
		t.Errorf("expected %v files to be copied, only %v copied", fcount, cnt)
	}
}

func TestCopyAllDirDstAbsent(t *testing.T) {
	tmp, src, fcount, err := createSourceDir(t)
	defer os.RemoveAll(tmp)

	// no destination directory
	dst := filepath.Join(tmp, "/dst")

	var cnt uint
	err = copyAll(src, dst, &cnt)
	if err != nil {
		t.Errorf("copy %q -> %q failed: %s", src, dst, err)
	}
	if int(cnt) != fcount {
		t.Errorf("expected %v files to be copied, only %v copied", fcount, cnt)
	}
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Errorf("expected %v to be created", dst)
	}
}

func TestCopyAllFileDstPresent(t *testing.T) {
	// create a workspace
	tmp, err := ioutil.TempDir("", "robotest-test")
	if err != nil {
		t.Skipf("unable to create tempdir: %s", err)
	}
	defer os.RemoveAll(tmp)

	// create source file
	fcount := 1
	src := filepath.Join(tmp, "/src")
	if err = ioutil.WriteFile(src, []byte("data"), 0640); err != nil {
		t.Skipf("unable to write to %s: %s", src, err)
	}

	// create destination directory
	dst := filepath.Join(tmp, "/dst")
	err = os.MkdirAll(dst, 0750)
	if err != nil {
		t.Skipf("unable to create dst dir: %s", err)
	}

	var cnt uint
	err = copyAll(src, dst, &cnt)
	if err != nil {
		t.Errorf("copy %q -> %q failed: %s", src, dst, err)
	}
	if int(cnt) != fcount {
		t.Errorf("expected %v files to be copied, only %v copied", fcount, cnt)
	}
}

func TestCopyAllFileDstAbsent(t *testing.T) {
	// create a workspace
	tmp, err := ioutil.TempDir("", "robotest-test")
	if err != nil {
		t.Skipf("unable to create tempdir: %s", err)
	}
	defer os.RemoveAll(tmp)

	// create source file
	fcount := 1
	src := filepath.Join(tmp, "/src")
	if err = ioutil.WriteFile(src, []byte("data"), 0640); err != nil {
		t.Skipf("unable to write to %s: %s", src, err)
	}

	// no destination directory
	dst := filepath.Join(tmp, "/dst")

	var cnt uint
	err = copyAll(src, dst, &cnt)
	if err != nil {
		t.Errorf("copy %q -> %q failed: %s", src, dst, err)
	}
	if int(cnt) != fcount {
		t.Errorf("expected %v files to be copied, only %v copied", fcount, cnt)
	}
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Errorf("expected %v to be created", dst)
	}

}
