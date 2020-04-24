package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brinick/fs"
)

func TestDepth(t *testing.T) {
	f, clean := newFile()
	defer clean()

	tests := []struct {
		name        string
		path1       string
		path2       string
		expectDepth int
		expectErr   error
	}{
		{"unrelated paths", "/random/root", "/unrelated/path", -1, nil},
		{"identical paths", "/random/root", "/random/root", 0, nil},
		{"inexistant path", "/random/root", "/random/root/missing.txt", 0, fs.InexistantError{"/random/root/missing.txt"}},
		{"real path", f.DirPath(), f.Path, 1, nil},
		{"real path subdir", f.DirPath(), f.Path, 1, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := fs.Depth(tt.path1, tt.path2)
			if tt.expectDepth != d {
				t.Errorf("%s: expected depth %d, got %d", tt.name, tt.expectDepth, d)
			}

			if tt.expectErr != err {
				t.Errorf("%s: expected error %v, got %v", tt.name, tt.expectErr, err)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	f, clean := newFile()
	defer clean()

	dstDir := filepath.Join(f.DirPath(), "subdir")
	if err := os.MkdirAll(dstDir, 0777); err != nil {
		t.Fatalf("unable to create dst subdir for copying: %v", err)
	}

	tests := []struct {
		name   string
		inSrc  string
		inDir  string
		expect error
	}{
		{"same file", f.Path, f.DirPath(), nil},
		{"inexistant src file", "/missing/file.txt", f.DirPath(), fs.InexistantError{"/missing/file.txt"}},
		{"inexistant dst dir", f.Path, "/missing/dir", fs.InexistantError{"/missing/dir"}},
		{"real copy", f.Path, dstDir, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fs.CopyFile(tt.inSrc, tt.inDir)
			if got != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, got)
			}
		})
	}
}

func TestIsFile(t *testing.T) {
	f, clean := newFile()
	defer clean()

	// Should be a normal file
	ok, err := fs.IsFile(f.Path)
	if err != nil {
		t.Errorf("unable to check if %s is a file: %v", f.Path, err)
	}

	if !ok {
		t.Errorf("%s: is a file, but claimed not to be", f.Path)
	}

	// Should not be a symlink
	ok, err = fs.IsSymLink(f.Path)
	if err != nil {
		t.Errorf("unable to check if %s is a symlink: %v", f.Path, err)
	}

	if ok {
		t.Errorf("%s: is a file, but claimed to be a symlink", f.Path)
	}
}

func TestIsLink(t *testing.T) {
	f, clean := newSymLink()
	defer clean()

	// Should be a symlink
	ok, err := fs.IsSymLink(f.Path)
	if err != nil {
		t.Errorf("unable to check if %s is a symlink: %v", f.Path, err)
	}

	if !ok {
		t.Errorf("%s: is a symlink, but claimed not to be", f.Path)
	}

	// Should not be a normal file
	ok, err = fs.IsFile(f.Path)
	if err != nil {
		t.Errorf("unable to check if %s is a normal file: %v", f.Path, err)
	}

	if ok {
		t.Errorf("%s: is a symlink file, but claimed to be a normal file", f.Path)
	}
}

func TestIsDir(t *testing.T) {
	d := tempDir()
	ok, err := fs.IsDir(d)
	if err != nil {
		t.Errorf("unable to check if %s is a directory: %v", d, err)
	}

	if !ok {
		t.Errorf("%s: is a directory, but claimed not to be", d)
	}
}

func TestPathExists(t *testing.T) {
	inexistant := "/inexistant/path"
	exists, err := fs.Exists(inexistant)
	if err != nil {
		t.Errorf("error checking if %s exists: %v", inexistant, err)
	}

	if exists {
		t.Errorf("%s: should not exist, but was marked as existing", inexistant)
	}

	f, clean := newFile()
	defer clean()

	parentDir := f.DirPath()

	exists, err = fs.Exists(parentDir)
	if err != nil {
		t.Errorf("error checking if %s exists: %v", parentDir, err)
	}

	if !exists {
		t.Errorf("%s: existing (temp) dir claimed to be inexistant", parentDir)
	}

	fpath := f.Path
	exists, err = fs.Exists(fpath)
	if err != nil {
		t.Errorf("error checking if %s exists: %v", fpath, err)
	}

	if !exists {
		t.Errorf("%s: should exist, but was marked as inexistant", fpath)
	}
}
