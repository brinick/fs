package fs_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/brinick/fs"
)

func newDir(t *testing.T, paths ...string) *fs.Directory {
	d, err := fs.NewDir(paths...)
	if err != nil {
		t.Fatalf("unable to create new Directory: %v", err)
	}

	return d
}

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

	tests := []struct {
		name     string
		patterns []string
		expect   bool
	}{
		{"no patterns", []string{}, false},
		{"single pattern char glob", []string{"blo?"}, true},
		{"single pattern multi glob", []string{"p*"}, true},
		{"single pattern multi glob", []string{"/*"}, true},
		{"multiple pattern fail", []string{"blipo", "blapo"}, false},
		{"single pattern spanning path frags", []string{"ip/bl"}, true},
		{"multiple pattern ok 1", []string{"blop", "blap", "nope"}, true},
		{"multiple pattern ok 2", []string{"bla", "blp", "b*p"}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matched, err := d.MatchAny(test.patterns...)
			if err != nil {
				t.Fatalf("unable to test for Directory Match: %v", err)
			}

			if matched != test.expect {
				t.Errorf(
					"Dir basename '%s' was match against '%s', expected match %t, got %t",
					d.Name(),
					strings.Join(test.patterns, ", "),
					test.expect,
					matched,
				)
			}
		})
	}
}

func TestMatchDir(t *testing.T) {
	d := newDir(t, "blip", "blap/blop", "blep")

	tests := []struct {
		name     string
		patterns []string
		expect   bool
	}{
		{"no patterns", []string{}, false},
		{"single pattern char glob", []string{"ble?"}, true},
		{"single pattern multi glob", []string{"b*"}, true},
		{"multiple pattern", []string{"blop", "blap", "nope"}, false},
		{"multiple pattern ok", []string{"bla", "blp", "b*p"}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matched, err := d.Match(test.patterns...)
			if err != nil {
				t.Fatalf("unable to test for Directory Match: %v", err)
			}

			if matched != test.expect {
				t.Errorf(
					"Dir basename '%s' was match against '%s', expected match %t, got %t",
					d.Name(),
					strings.Join(test.patterns, ", "),
					test.expect,
					matched,
				)
			}
		})
	}
}
