package fs_test

import (
	"path/filepath"
	"testing"

	"github.com/brinick/fs"
)

func TestCreateCurrentDir(t *testing.T) {
	d, err := fs.NewDir()
	if err != nil {
		t.Fatalf("unable to create NewDir: %v", err)
	}

	base := filepath.Base(d.Path)
	if base != "fs" {
		t.Errorf("directory expected with base path fs, got %s", base)
	}
}

func TestMatchAnyDir(t *testing.T) {
	d := newDir(t, "blip", "blap/blop", "blep")
	matched, err := d.MatchAny("blip/")
	if err != nil {
		t.Fatalf("unable to test for Directory Match: %v", err)
	}

	if !matched {
		t.Errorf("Directory %s should match blap, but did not", d.Path)
	}
}

func TestMatchDir(t *testing.T) {
	d := newDir(t, "blip", "blap/blop", "blep")
	patt := "ble?"
	matched, err := d.Match(patt)
	if err != nil {
		t.Fatalf("unable to test for Directory Match: %v", err)
	}

	if !matched {
		t.Errorf("Directory with base name '%s' should match patt '%s', but did not", d.Name(), patt)
	}
}

func newDir(t *testing.T, paths ...string) *fs.Directory {
	d, err := fs.NewDir(paths...)
	if err != nil {
		t.Fatalf("unable to create new Directory: %v", err)
	}

	return d
}
