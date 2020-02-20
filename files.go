package fs

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// File returns a new file instance for the given path
func File(path string) *file {
	return &file{}
}

// file represents a file or symlink
type file struct {
	Path string
}

// Dir returns the file's parent directory
func (f *file) Dir() *Directory {
	return &Directory{filepath.Dir(f.Path)}
}

// Match returns a boolean to indicate if any of the provided patterns
// match against the file's name
func (f file) Match(patterns ...string) (bool, error) {
	name := f.Name()
	for _, patt := range patterns {
		ok, err := filepath.Match(patt, name)
		if err != nil {
			return false, err
		}

		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (f *file) WriteLines(lines []string, flag int, perm os.FileMode) error {
	fd, err := os.OpenFile(f.Path, flag, perm)
	if err != nil {
		return err
	}

	defer fd.Close()

	for _, line := range lines {
		_, err = fd.WriteString(line + "\n")
		if err != nil {
			break
		}
	}

	return nil
}

func (f *file) Write(data []byte, flag int, perm os.FileMode) error {
	fd, err := os.OpenFile(f.Path, flag, perm)
	if err != nil {
		return err
	}

	defer fd.Close()
	_, err = fd.Write(data)
	return err
}

func (f *file) Bytes() ([]byte, error) {
	exists, err := f.Exists()
	if err != nil {
		return []byte{}, err
	}
	if !exists {
		return []byte{}, fmt.Errorf("%s: file inexistant", f.Path)
	}
	return ioutil.ReadFile(f.Path)
}

// Lines returns the file contents as a slice of lines
func (f *file) Lines() ([]string, error) {
	var lines = []string{}

	exists, err := f.Exists()
	if err != nil {
		return lines, err
	}
	if !exists {
		return lines, fmt.Errorf("%s: file inexistant", f.Path)
	}

	fd, err := os.Open(f.Path)
	if err != nil {
		return lines, err
	}
	defer fd.Close()

	s := bufio.NewScanner(fd)
	for s.Scan() {
		lines = append(lines, s.Text())
	}

	return lines, s.Err()

}

// Text returns the file contents as a string
func (f *file) Text() (string, error) {
	lines, err := f.Lines()
	if err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

// Name returns the base part of the file path
func (f *file) Name() string {
	return filepath.Base(f.Path)
}

// NameExt returns the file name split into name and extension
func (f *file) NameExt() (string, string) {
	toks := strings.Split(f.Name(), ".")
	ntoks := len(toks)
	if ntoks == 1 {
		return toks[0], ""
	}
	if ntoks == 2 {
		return toks[0], toks[1]
	}

	last := len(toks) - 1
	return strings.Join(toks[:last], "."), toks[last]
}

// Exists checks if the given file path exists
func (f *file) Exists() (bool, error) {
	return Exists(f.Path)
}

// Size returns the size in bytes of the file
func (f *file) Size() int64 {
	if exists, _ := f.Exists(); exists {
		if info, err := os.Stat(f.Path); err == nil {
			return info.Size()
		}
	}

	return 0
}

// CopyTo will copy the file to the given destination directory
func (f *file) CopyTo(dst string) error {
	return CopyFile(f.Path, dst)
}

// Resolve will resolve the symbolic link, if it is one.
// Otherwise it will just return the file path.
func (f *file) Resolve() (string, error) {
	isLink, err := f.IsSymLink()
	if err != nil {
		return "", err
	}

	if !isLink {
		return f.Path, nil
	}

	tgt, err := os.Readlink(f.Path)
	if err != nil {
		return "", err
	}

	return filepath.Abs(tgt)
}

// IsSymLink checks if the file is a symlink
func (f *file) IsSymLink() (bool, error) {
	return IsSymLink(f.Path)
}

// ------------------------------------------------------------------

// Files represents a collection of files, some of which may be symlinks
type Files []*file

// Paths returns the list of file paths for each file
func (f *Files) Paths() []string {
	var paths []string
	for _, ff := range *f {
		paths = append(paths, ff.Path)
	}

	return paths
}

// Symlinks returns those files which are symlinks.
func (f *Files) Symlinks() (*Files, error) {
	var links Files
	for _, file := range *f {
		isLink, err := file.IsSymLink()
		if err != nil {
			return nil, err
		}
		if isLink {
			links = append(links, file)
		}
	}

	return &links, nil
}

// Resolve returns a list of absolute paths to the
// targets of any symlinks in the files. Empty list if
// there are no symlinks.
func (f *Files) Resolve() ([]string, error) {
	var paths []string
	links, err := f.Symlinks()
	if err != nil {
		return paths, err
	}

	for _, link := range *links {
		path, err := link.Resolve()
		if err != nil {
			return paths, err
		}

		paths = append(paths, path)
	}

	return paths, err
}

// Names returns the list of names of all the files
func (f *Files) Names() []string {
	var names []string
	for _, ff := range *f {
		names = append(names, ff.Name())
	}

	return names
}

// Match returns the subset of Files whose name matches
// against one or more of the given glob patterns
func (f *Files) Match(patterns ...string) (*Files, error) {
	return filesMatcher(f, true, patterns...)
}

// NotMatch returns the subset of Files whose name
// does not match against any of the given glob patterns
func (f *Files) NotMatch(patterns ...string) (*Files, error) {
	return filesMatcher(f, false, patterns...)
}

// Remove will delete files matching the given glob patterns
func (f *Files) Remove(patterns ...string) error {
	matches, err := f.Match(patterns...)
	if err != nil {
		return err
	}

	for _, m := range *matches {
		if err := os.RemoveAll(m.Path); err != nil {
			return fmt.Errorf("unable to delete dir tree at %s (%w)", m.Path, err)
		}
	}

	return nil
}

// RemoveFiles will delete files matching the given file name glob,
// found at most maxDepth directories below startDir
func RemoveFiles(startDir, fileNameGlob string, maxDepth int, ignore []string) error {
	files, err := FindFiles(startDir, fileNameGlob, maxDepth, ignore)
	if err != nil {
		return err
	}

	for _, file := range files {
		os.Remove(file)
	}

	return nil
}

// FindFiles finds all files matching a given file name glob, or exact name,
// below the given start directory. The search goes at most max depth
// directories down.
func FindFiles(startDir, fileNameGlob string, maxDepth int, ignore []string) ([]string, error) {
	_, files, err := WalkTree(startDir, ignore, maxDepth)
	var matches []string
	for _, f := range files {
		matched, _ := filepath.Match(fileNameGlob, filepath.Base(f))
		if matched {
			matches = append(matches, f)
		}
	}
	return matches, err
}

type acceptFunc func(string) (bool, error)

// FindIf has the same signature as Find but returns only files
// that return true from the accept function
func FindIf(startDir, fileNameGlob string, maxDepth int, ignore []string, accept acceptFunc) ([]string, error) {
	matches, err := FindFiles(startDir, fileNameGlob, maxDepth, ignore)

	if err != nil {
		return nil, err
	}

	if accept == nil {
		return matches, nil
	}

	// Use the same backing array for the filtered matches
	accepted := matches[:0]
	for _, m := range matches {
		if ok, err := accept(m); ok && err == nil {
			accepted = append(accepted, m)
		}
	}

	return accepted, nil
}
