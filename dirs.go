package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// NewDir creates a new directory instance comprised of the
// joining of the provided paths. If no paths are provided,
// the current directory is returned.
func NewDir(paths ...string) (*Directory, error) {
	if len(paths) == 0 {
		d, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		return &Directory{d}, nil
	}

	return &Directory{
		Path: filepath.Join(paths...),
	}, nil
}

// Directory represents a particular directory
type Directory struct {
	Path string
}

// Match returns a boolean to indicate if any of the provided patterns
// match against the directory's name
func (d *Directory) Match(patterns ...string) (bool, error) {
	name := d.Name()
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

// Exists checks if this directory path exists and is a directory
func (d *Directory) Exists() (bool, error) {
	return IsDir(d.Path)
}

// Dir returns the parent path of the current directory
func (d *Directory) Dir() string {
	return filepath.Dir(d.Path)
}

// Name returns the base path of the current directory
func (d *Directory) Name() string {
	return filepath.Base(d.Path)
}

// Join returns a new Directory instance with a path
// created by joining current directory with the sub dirs
// passed in. If the path does not exist, or if there is
// an error trying to find out, the returned value is nil.
func (d *Directory) Join(frags ...string) *Directory {
	path := filepath.Join(d.Path, strings.Join(frags, "/"))
	var cd *Directory
	if ok, _ := Exists(path); ok {
		cd = &Directory{
			Path: path,
		}
	}
	return cd
}

// Append is like Join except that it does not check if the
// resulting file path actually exists.
func (d *Directory) Append(frags ...string) *Directory {
	path := filepath.Join(d.Path, strings.Join(frags, "/"))
	return &Directory{
		Path: path,
	}
}

// Create will create the given directory path, including
// missing intermediate dirs, if inexistant.
func (d *Directory) Create(mode os.FileMode) error {
	exists, err := d.Exists()
	if err != nil {
		return err
	}

	if !exists {
		return os.MkdirAll(d.Path, mode)
	}

	return nil
}

// CopyTo recursively copies the content of the directory
// to the path rooted at the given directory. If the destination
// already exists, an error is returned and no copy is performed.
func (d *Directory) CopyTo(dst string) error {
	var (
		err     error
		fds     []os.FileInfo
		srcinfo os.FileInfo
		exists  bool
	)

	dstDir := Directory{dst}
	exists, err = dstDir.Exists()
	if err != nil {
		return fmt.Errorf(
			"unable to check if CopyTo destination dir (%s) exists already (%w)",
			dst,
			err,
		)
	}

	if exists {
		return fmt.Errorf("cannot copy to an existing destination dir (%s)", dst)
	}

	if srcinfo, err = os.Stat(d.Path); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(d.Path); err != nil {
		return err
	}

	for _, fd := range fds {
		srcfp := filepath.Join(d.Path, fd.Name())
		dstfp := filepath.Join(dst, fd.Name())

		if fd.IsDir() {
			d, err := NewDir(srcfp)
			if err != nil {
				return fmt.Errorf(
					"failed to create Directory instance for %s (%v)",
					srcfp,
					err,
				)
			}

			if err = d.CopyTo(dstfp); err != nil {
				return fmt.Errorf("cannot copy dir %s to %s: %w", srcfp, dstfp, err)
			}
		} else {
			if err = CopyFile(srcfp, dst); err != nil {
				return fmt.Errorf("cannot copy file %s to dir %s (%w)", srcfp, dst, err)
			}
		}
	}

	return nil
}

// SubDirs returns a list of Directory instances for all directories
// within the current directory that match at least one of the
// provided glob patterns. If no patterns are provided, match all.
func (d *Directory) SubDirs(patterns ...string) (*Directories, error) {
	list, err := dirLister(d.Path)
	if err != nil {
		return nil, err
	}

	dirs, err := list.dirs()
	if err != nil {
		return nil, err
	}

	return dirs.Match(patterns...), nil
}

// Files returns a Files instance containing the list of
// files, excluding symlinks, within the current directory that match at least
// one of the provided glob patterns. If no patterns are
// provided, all files are matched.
func (d *Directory) Files(patterns ...string) (*Files, error) {
	entries, err := dirLister(d.Path)
	if err != nil {
		return nil, err
	}

	files, err := entries.files(false)
	if err != nil {
		return nil, err
	}

	matches, err := files.Match(patterns...)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

// FilesAll is the same as Files(), except that the
// returned list includes symbolic links.
func (d *Directory) FilesAll(patterns ...string) (*Files, error) {
	entries, err := dirLister(d.Path)
	if err != nil {
		return nil, err
	}
	files, err := entries.files(true)
	if err != nil {
		return nil, err
	}

	matches, err := files.Match(patterns...)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

// Symlinks returns the symbolic links in the directory
func (d *Directory) Symlinks(patterns ...string) (*Files, error) {
	return nil, nil
}

// Remove will delete the directory
func (d *Directory) Remove() error {
	return os.RemoveAll(d.Path)
}

// ------------------------------------------------------------------

// Dirs returns a Directories instance for the given dirs
func Dirs(dirs ...string) *Directories {
	var d Directories
	for _, dir := range dirs {
		dd, _ := NewDir(dir)
		d = append(d, dd)
	}

	return &d
}

// Directories represents a collection of directory instances
type Directories []*Directory

// Names returns the list of names of each base part
// of the directories
func (d *Directories) Names() []string {
	var names []string
	for _, subdir := range *d {
		names = append(names, subdir.Name())
	}

	return names
}

// Match returns the subset of directories whose base name matches
// against any of the given glob patterns. If no patterns are supplied,
// the operation is a no-op and the same Directories instance is returned.
func (d *Directories) Match(patterns ...string) *Directories {
	if len(patterns) == 0 {
		return d
	}

	var newD Directories
	for _, pattern := range patterns {
		for _, dd := range *d {
			if ok, _ := filepath.Match(pattern, dd.Name()); ok {
				newD = append(newD, dd)
			}
		}
	}

	return &newD
}

// NotMatch returns the subset of Directories whose name
// does not match against the given glob pattern. If no patterns
// are supplied, an empty Directories instance is returned.
func (d *Directories) NotMatch(patterns ...string) *Directories {
	if len(patterns) == 0 {
		return &Directories{}
	}
	var newD Directories
	for _, pattern := range patterns {
		for _, dd := range *d {
			if ok, _ := filepath.Match(pattern, dd.Name()); !ok {
				newD = append(newD, dd)
			}
		}
	}

	return &newD
}

// Remove will delete the directories
func (d *Directories) Remove() error {
	for _, dir := range *d {
		if err := dir.Remove(); err != nil {
			return err
		}
	}
	return nil
}
