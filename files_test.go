package fs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brinick/fs"
)

func tempDir() string {
	dir, err := ioutil.TempDir("", "fs_files_test")
	if err != nil {
		panic(
			fmt.Sprintf(
				"unable to make a temporary directory: %v",
				err,
			),
		)
	}

	return dir
}

type cleanUpFn func()

func newFileInDir(dir string) *fs.File {
	f := fs.NewFile(filepath.Join(dir, "test.file.txt"))
	if err := f.Touch(false); err != nil {
		panic(fmt.Sprintf("unable to touch new file %s: %v", f.Path, err))
	}
	return f
}

func newFile() (*fs.File, cleanUpFn) {
	file := newFileInDir(tempDir())
	cleanup := func() {
		os.RemoveAll(file.DirPath())
	}

	return file, cleanup
}

func newSymLink() (*fs.File, cleanUpFn) {
	f, clean := newFile()
	link := filepath.Join(f.DirPath(), "symlink")
	if err := os.Symlink(f.Path, link); err != nil {
		panic(fmt.Sprintf("cannot make a symlink! %v", err))
	}
	return fs.NewFile(link), clean
}

func TestGetFileDir(t *testing.T) {
	parentDir := tempDir()
	defer os.RemoveAll(parentDir)
	fdir := newFileInDir(parentDir).Dir().Path
	if fdir != parentDir {
		t.Errorf("expected file parent dir to be %s, got %s", parentDir, fdir)
	}
}

func TestModTime(t *testing.T) {
	f, clean := newFile()
	defer clean()

	when, err := f.ModTime()
	if err != nil {
		t.Errorf("unable to get file modtime: %v", err)
	}

	now := time.Now()
	diff := now.Sub(*when)
	switch {
	case diff < 0:
		t.Errorf(
			"just created file has modtime (%s) more recent than now (%s)",
			*when,
			now,
		)
	case diff > 1*time.Second:
		t.Errorf(
			"just created file has modtime (%s) "+
				"older than 1 sec compared to now (%s)",
			*when,
			now,
		)
	}
}

func TestMatchFileName(t *testing.T) {
	f, clean := newFile()
	defer clean()

	type expect struct {
		matched bool
		err     error
	}

	tests := map[string]struct {
		in  []string
		out expect
	}{
		"exact file":                           {[]string{"test.file.txt"}, expect{true, nil}},
		"partial match":                        {[]string{"test.fi*"}, expect{true, nil}},
		"partial match 2":                      {[]string{"?est.fi*"}, expect{true, nil}},
		"empty string":                         {[]string{""}, expect{false, nil}},
		"single space":                         {[]string{" "}, expect{false, nil}},
		"asterisk":                             {[]string{"*"}, expect{true, nil}},
		"asterisk with leading/trailing space": {[]string{" * "}, expect{true, nil}},
		"empty then asterisk":                  {[]string{" ", "*"}, expect{true, nil}},
	}

	for name, tt := range tests {
		gotB, gotE := f.Match(tt.in...)
		if gotE != tt.out.err {
			t.Errorf(
				"%s: different error values. Expect %v, got %v",
				name,
				tt.out.err,
				gotE,
			)
		}

		if gotB != tt.out.matched {
			t.Errorf(
				"%s: different matched values. Expect %t, got %t",
				name,
				tt.out.matched,
				gotB,
			)
		}
	}
}

func TestSetFileMode(t *testing.T) {
	f, clean := newFile()
	defer clean()

	if err := f.SetFileMode(0777); err != nil {
		t.Fatalf("unable to set file mode: %v", err)
	}

	mode, err := f.FileMode()
	if err != nil {
		t.Fatalf("set file mode, but unable to read it back: %v", err)
	}
	if mode.Perm() != 0777 {
		t.Errorf("incorrect file mode, expected 0777, got %v", mode)
	}
}

func TestTouchFile(t *testing.T) {
	f, clean := newFile()
	defer clean()

	startMode, err := f.FileMode()
	if err != nil {
		t.Fatalf("unable to get file mode: %v", err)
	}

	startModTime, err := f.ModTime()
	if err != nil {
		t.Fatalf("unable to get start modtime: %v", err)
	}

	// briefly sleep to allow for checking modtime in a bit
	time.Sleep(100 * time.Millisecond)

	for _, ignoreIfExists := range []bool{true, false} {
		if err := f.Touch(ignoreIfExists); err != nil {
			t.Fatalf("unable to 'touch' file: %v", err)
		}

		info, err := os.Stat(f.Path)
		if err != nil {
			t.Fatalf("unable to stat file %s: %v", f.Path, err)
		}

		if info.Mode() != startMode {
			t.Errorf("file mode changed but should not have. Expected %v, got %v", startMode, info.Mode())
		}

		nowModTime, err := f.ModTime()
		if err != nil {
			t.Fatalf("unable to get start modtime: %v", err)
		}

		changed := !startModTime.Equal(*nowModTime)
		shouldHaveChanged := (!ignoreIfExists)
		if shouldHaveChanged && !changed {
			t.Errorf("file modtime did not change, but should have. Expected %v, got %v", nowModTime, startModTime)
		} else if !shouldHaveChanged && changed {
			t.Errorf("file modtime changed, but should not have. Expected %v, got %v", startModTime, nowModTime)
		}
	}
}

func TestReadFile(t *testing.T) {
	f, clean := newFile()
	defer clean()

	output := []string{"line1", "line2"}
	args := []string{
		fmt.Sprintf("'%s' >& %s", strings.Join(output, "\n"), f.Path),
	}
	err := exec.Command("echo", args...).Run()
	if err != nil {
		t.Fatalf("failed to cat text to file: %v", err)
	}

	lines, err := f.Lines()
	if err != nil {
		t.Fatalf("unable to read file lines: %v", err)
	}

	for i, line := range lines {
		if line != output[i] {
			t.Errorf("file lines mismatch, expected: %s\ngot: %s", output[i], line)
		}
	}

}

func TestWriteThenReadFile(t *testing.T) {
	f, clean := newFile()
	defer clean()

	output := []string{"hello", "world"}
	if err := f.WriteLines(output); err != nil {
		t.Fatalf("unable to write lines to file: %v", err)
	}

	lines, err := f.Lines()
	if err != nil {
		t.Fatalf("unable to read file lines: %v", err)
	}

	for i, line := range lines {
		if line != output[i] {
			t.Errorf("file lines mismatch, expected: %s\ngot: %s", output[i], line)
		}
	}
}

func TestAppendLines(t *testing.T) {
	f, clean := newFile()
	defer clean()

	first := []string{"hello", "world"}
	if err := f.AppendLines(first); err != nil {
		t.Errorf("unable to append lines: %v", err)
	}

	checkFileHasLines(t, f, first)

	second := []string{"it", "is", "I"}
	if err := f.AppendLines(second); err != nil {
		t.Errorf("unable to append lines: %v", err)
	}
	checkFileHasLines(t, f, append(first, second...))
}

func checkFileHasLines(t *testing.T, f *fs.File, expect []string) {
	lines, err := f.Lines()
	if err != nil {
		t.Fatalf("unable to read file lines: %v", err)
	}

	if len(expect) != len(lines) {
		t.Fatalf("file should contain %d lines, instead has %d", len(expect), len(lines))
	}

	for i, line := range lines {
		if line != expect[i] {
			t.Errorf("line #%d, expected %s, got %s", i, expect[i], line)
		}
	}
}
