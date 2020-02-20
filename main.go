// Package fs is a set of file system utilities
package fs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Exists checks if the given path exists.
// It may be a directory, normal file or symlink.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	// We return false, however that may not be correct.
	// The point is that as we have an error, we can't
	// really know if the path exists.
	return false, err
}

// IsSymLink checks if the given path is a symlink
func IsSymLink(path string) (bool, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return (fi.Mode()&os.ModeSymlink != 0), nil
}

// IsDir checks if the given path is a directory
func IsDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fi.IsDir(), nil
}

// IsFile checks if the given path is a normal file
func IsFile(path string) (bool, error) {
	if ok, err := IsDir(path); ok || err != nil {
		return false, err
	}
	if ok, err := IsSymLink(path); ok || err != nil {
		return false, err
	}

	return true, nil
}

// ------------------------------------------------------------------

// entries is the list of items in a directory
// a mixture of dirs, files, symlinks
type entries struct {
	dir    string
	values []os.FileInfo
}

func (e *entries) dirs() (*Directories, error) {
	var dirs Directories
	for _, entry := range e.values {
		if entry.IsDir() {
			fullpath := filepath.Join(e.dir, entry.Name())
			dirs = append(dirs, &Directory{Path: fullpath})
		}
	}

	return &dirs, nil
}

func (e *entries) files(includeSymLinks bool) (*Files, error) {
	var files Files
	for _, entry := range e.values {
		if entry.IsDir() {
			continue
		}

		fullpath := filepath.Join(e.dir, entry.Name())
		if !includeSymLinks {
			isSym, err := IsSymLink(fullpath)
			if err != nil {
				return nil, fmt.Errorf("unable to check if file is symlink %s (%w)", fullpath, err)
			}

			if isSym {
				continue
			}
		}

		files = append(files, &file{Path: fullpath})
	}

	return &files, nil
}

func (e *entries) symlinks() (*Files, error) {
	var files Files
	for _, entry := range e.values {
		if entry.IsDir() {
			continue
		}

		fullpath := filepath.Join(e.dir, entry.Name())
		isSym, err := IsSymLink(fullpath)
		if err != nil {
			return nil, fmt.Errorf("unable to check if file is symlink %s (%w)", fullpath, err)
		}

		if isSym {
			files = append(files, &file{Path: fullpath})
		}
	}

	return &files, nil
}

func (e *entries) filesAll() (*Files, error) {
	return e.files(true)
}

// ------------------------------------------------------------------

// dirsMatcher returns the subset of Directories that, depending on the
// shouldFind boolean, match or do not match the provided pattern.
func dirsMatcher(dirs *Directories, shouldFind bool, patterns ...string) (*Directories, error) {
	if len(patterns) == 0 {
		if shouldFind {
			return dirs, nil
		}

		return nil, nil
	}

	var matches Directories
	for _, dir := range *dirs {
		ok, err := dir.Match(patterns...)
		if err != nil {
			return nil, err
		}
		if ok == shouldFind {
			matches = append(matches, dir)
		}
	}

	return &matches, nil
}

// filesMatcher returns the subset of ... that, depending on the
// shouldFind boolean, match or do not match the provided pattern.
func filesMatcher(files *Files, shouldFind bool, patterns ...string) (*Files, error) {
	if len(patterns) == 0 {
		if shouldFind {
			return files, nil
		}

		return nil, nil
	}

	var matches Files
	for _, file := range *files {
		ok, err := file.Match(patterns...)
		if err != nil {
			return nil, err
		}

		if ok == shouldFind {
			matches = append(matches, file)
		}
	}

	return &matches, nil
}

func dirLister(dir string) (*entries, error) {
	entriesList, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	return &entries{dir: dir, values: entriesList}, nil
}

// ------------------------------------------------------------------

// Depth returns the integer number of directories that
// path is below root. If root is not a prefix of path, it
// returns 0. If path is a file, the depth is calculated with
// respect to the parent directory of the file.
func Depth(root, path string) (int, error) {
	removeTrailingSlash := func(s string) string {
		if strings.HasSuffix(s, "/") {
			s = s[:len(s)-1]
		}

		s, _ = filepath.Abs(s)
		return s
	}

	root = removeTrailingSlash(root)
	path = removeTrailingSlash(path)

	if root == path {
		return 0, nil
	}

	if !strings.HasPrefix(path, root) {
		return 0, fmt.Errorf("%s not a prefix of %s", root, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	if !info.IsDir() {
		path = filepath.Dir(path)
	}

	path = strings.Replace(path, root, "", 1)
	path = strings.Trim(path, "/")
	dirs := strings.Split(path, "/")
	return len(dirs), nil
}

// TreeSize walks the tree starting at root directory,
// and totals the size of all files it finds. Directories
// matching entries in the excludeDirs list are not traversed.
// The grand total in bytes is returned.
func TreeSize(root string, excludeDirs []string) (int64, error) {
	totSize := int64(0)
	err := filepath.Walk(
		root,
		func(path string, pathInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if pathInfo.IsDir() {
				for _, e := range excludeDirs {
					if pathInfo.Name() == e {
						return filepath.SkipDir
					}
				}
			} else {
				totSize += pathInfo.Size()
			}

			return nil
		},
	)

	return totSize, err
}

// WalkTree walks the tree starting from root, returning
// all directories and files found. If maxDepth is > 0,
// the walk will truncate this many levels below root dir.
// Directories in the excludeDirs slice will be ignored.
func WalkTree(root string, excludeDirs []string, maxdepth int) ([]string, []string, error) {
	dirs := []string{}
	files := []string{}

	currDepth := func(path string) int {
		depth, _ := Depth(root, path)
		return depth
	}

	err := filepath.Walk(
		root,
		func(path string, pathInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !pathInfo.IsDir() {
				files = append(files, path)
			} else {
				if maxdepth > 0 && currDepth(path) > maxdepth {
					return filepath.SkipDir
				}

				for _, e := range excludeDirs {
					if pathInfo.Name() == e {
						return filepath.SkipDir
					}
				}

				dirs = append(dirs, path)
			}

			return nil
		},
	)

	return dirs, files, err
}

// CopyFile copies the src file to the dst directory, giving the
// destination file the same file mode permissions as the source.
func CopyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("unable to open input file %s for reading (%w)", src, err)
	}

	defer source.Close()

	sourceFI, err := source.Stat()
	if err != nil {
		return err
	}

	fname := filepath.Join(dst, filepath.Base(src))
	dest, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY, sourceFI.Mode())
	if err != nil {
		return err
	}

	defer dest.Close()

	_, err = io.Copy(source, dest)
	return err
}
