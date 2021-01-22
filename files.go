package fs

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NewFile returns a new file instance for the given path
func NewFile(path string) *File {
	return &File{
		Path: path,
	}
}

// File represents a file or symlink
type File struct {
	Path string
}

// Dir returns the file's parent Directory
func (f *File) Dir() *Directory {
	return &Directory{filepath.Dir(f.Path)}
}

// DirPath returns the file's parent Directory path
func (f *File) DirPath() string {
	return f.Dir().Path
}

// ModTime returns the last modification time of this file
func (f *File) ModTime() (*time.Time, error) {
	info, err := os.Stat(f.Path)
	if err != nil {
		return nil, err
	}

	mt := info.ModTime()
	return &mt, nil
}

// Match returns a boolean to indicate if any of the provided patterns
// match against the file's name
func (f *File) Match(patterns ...string) (bool, error) {
	name := f.Name()
	for _, patt := range patterns {
		patt = strings.TrimSpace(patt)
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

// SetFileMode changes the mode of the file
func (f *File) SetFileMode(perm os.FileMode) error {
	return os.Chmod(f.Path, perm)
}

// FileMode gets the file mode if it exists, else returns an error
func (f *File) FileMode() (os.FileMode, error) {
	var mode os.FileMode
	fi, err := os.Stat(f.Path)
	if err != nil {
		return mode, err
	}

	return fi.Mode(), nil
}

// Create will create the file with default file permission.
// It will truncate the file if it already exists.
func (f *File) Create() error {
	return f.CreateWithPerm(0000) // set the default mode
}

// CreateWithPerm will create the file with the given permission.
// It will truncate the file if it already exists.
func (f *File) CreateWithPerm(perm os.FileMode) error {
	fd, err := os.Create(f.Path)
	if err != nil {
		return fmt.Errorf("unable to create file: %v", err)
	}
	defer fd.Close()

	if perm != 0000 {
		if err = fd.Chmod(perm); err != nil {
			return fmt.Errorf("unable to change file mode: %v", err)
		}
	}
	return nil
}

// AppendLines appends the given lines to the file contents.
// If the file does not exist, an error is returned.
func (f *File) AppendLines(lines []string) error {
	return f.writeLines(lines, true)
}

// WriteLines writes the given lines to the file.
// If the file does not exist, an error is returned.
func (f *File) WriteLines(lines []string) error {
	return f.writeLines(lines, false)
}

// Write writes the given data bytes to the file.
// If the file does not exist, an error is returned.
func (f *File) Write(data []byte) error {
	return f.writeBytes(data, false)
}

// Append writes the given data bytes to the end of the file.
// If the file does not exist, an error is returned.
func (f *File) Append(data []byte) error {
	return f.writeBytes(data, true)
}

// Bytes returns the file content as a slice of bytes
func (f *File) Bytes() ([]byte, error) {
	exists, err := f.Exists()
	if err != nil {
		return []byte{}, err
	}
	if !exists {
		return []byte{}, InexistantError{f.Path}
	}
	return ioutil.ReadFile(f.Path)
}

// Lines returns the file contents as a slice of lines/strings
func (f *File) Lines() ([]string, error) {
	var lines = []string{}

	exists, err := f.Exists()
	if err != nil {
		return lines, err
	}
	if !exists {
		return lines, InexistantError{f.Path}
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

// Touch will create an empty file if it is inexistant, else will update
// the last modified and access times. If ignoreIfExists is True, then
// this update will not occur.
func (f *File) Touch(ignoreIfExists bool) error {
	var (
		err    error
		exists bool
	)

	exists, err = f.Exists()
	if err != nil {
		return err
	}

	if ignoreIfExists && exists {
		return nil
	}

	if !exists {
		if err := f.Create(); err != nil {
			return err
		}
	}

	// touch the existing file, update access/mod times
	now := time.Now().Local()
	return os.Chtimes(f.Path, now, now)
}

// Text returns the file contents as a string
func (f *File) Text() (string, error) {
	lines, err := f.Lines()
	if err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

// Name returns the base part of the file path
func (f *File) Name() string {
	return filepath.Base(f.Path)
}

// NameExt returns the file name split into name and extension
func (f *File) NameExt() (string, string) {
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
func (f *File) Exists() (bool, error) {
	return Exists(f.Path)
}

// Size returns the size in bytes of the file
func (f *File) Size() int64 {
	if exists, _ := f.Exists(); exists {
		if info, err := os.Stat(f.Path); err == nil {
			return info.Size()
		}
	}

	return 0
}

// RenameTo renames the current file to the new path. If the destination
// directory does not exist an error is returned.
func (f *File) RenameTo(newpath string) error {
	err := os.Rename(f.Path, newpath)
	if err == nil {
		// update this File struct if no error occured
		f.Path = newpath
	}
	return err
}

// ExportTo creates a copy of the file at the given path.
func (f *File) ExportTo(copypath string) error {
	var err error
	copyDir := filepath.Dir(copypath)
	if copyDir == f.DirPath() {
		// Need to use a tempdir if the parent dirs are the same
		copyDir, err = ioutil.TempDir("", "")
		if err != nil {
			return fmt.Errorf("unable to create temp dir: %v", err)
		}
	}
	// 1. Copy the current file to the copyDir
	if err := CopyFile(f.Path, copyDir); err != nil {
		return err
	}

	// 2. Rename the file to the Base(copypath)
	return os.Rename(filepath.Join(copyDir, f.Name()), copypath)
}

// CopyTo copies the file to the given destination directory.
// If the destination and the file directory are the same, nothing happens
// and no error is returned.
func (f *File) CopyTo(dstDir string) error {
	return CopyFile(f.Path, dstDir)
}

// Backup copies the file to the same directory and adds a .bck suffix.
func (f *File) Backup() error {
	bckup := f.Name() + ".bck"
	return f.ExportTo(filepath.Join(f.DirPath(), bckup))
}

// Recover looks for a file in the same directory with .bck suffix
// and overwrites the file with this backup file.
func (f *File) Recover() error {
	bckup := f.Name() + ".bck"
	bckup = filepath.Join(f.DirPath(), bckup)
	ok, err := Exists(bckup)
	if err != nil {
		return fmt.Errorf("unable to check if backup file exists: %v", err)
	}
	if !ok {
		return fmt.Errorf("backup file %s does not exist, nothing to recover", bckup)
	}

	return os.Rename(bckup, f.Path)
}

// Resolve will resolve the symbolic link, if it is one.
// Otherwise it will just return the file path.
func (f *File) Resolve() (string, error) {
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
func (f *File) IsSymLink() (bool, error) {
	return IsSymLink(f.Path)
}

func (f *File) isInexistant() bool {
	_, err := os.Stat(f.Path)
	return os.IsNotExist(err)
}

func (f *File) open(flag int) (*os.File, error) {
	// Stop if the file does not exist
	if f.isInexistant() {
		return nil, InexistantError{f.Path}
	}

	// Get the file's current file mode
	perm, err := f.FileMode()
	if err != nil {
		return nil, fmt.Errorf("unable to get file mode: %v", err)
	}

	return os.OpenFile(f.Path, flag, perm)
}

func (f *File) writeBytes(data []byte, append bool) error {
	flag := os.O_WRONLY
	if append {
		flag |= os.O_APPEND
	}

	fd, err := f.open(flag)
	if err != nil {
		return err
	}

	_, err = fd.Write(data)
	return err
}

func (f *File) writeLines(lines []string, append bool) error {
	flag := os.O_WRONLY
	if append {
		flag |= os.O_APPEND
	}

	fd, err := f.open(flag)
	if err != nil {
		return err
	}

	defer fd.Close()

	for _, line := range lines {
		if _, err := fd.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return nil

}

// ------------------------------------------------------------------

// Files represents a collection of files, some of which may be symlinks
type Files []*File

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
