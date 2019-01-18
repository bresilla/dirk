package dirk

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

type Dirent struct {
	name     string
	modeType os.FileMode
}

func NewDirent(osPathname string) (*Dirent, error) {
	fi, err := os.Lstat(osPathname)
	if err != nil {
		return nil, errors.Wrap(err, "cannot lstat")
	}
	return &Dirent{
		name:     filepath.Base(osPathname),
		modeType: fi.Mode() & os.ModeType,
	}, nil
}
func (de Dirent) Name() string          { return de.name }
func (de Dirent) ModeType() os.FileMode { return de.modeType }
func (de Dirent) IsDir() bool           { return de.modeType&os.ModeDir != 0 }
func (de Dirent) IsRegular() bool       { return de.modeType&os.ModeType == 0 }
func (de Dirent) IsSymlink() bool       { return de.modeType&os.ModeSymlink != 0 }

type Dirents []*Dirent

func (l Dirents) Len() int           { return len(l) }
func (l Dirents) Less(i, j int) bool { return l[i].name < l[j].name }
func (l Dirents) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// DefaultScratchBufferSize specifies the size of the scratch buffer that will
// be allocated by Walk, ReadDirents, or ReadDirnames when a scratch buffer is
// not provided or the scratch buffer that is provided is smaller than
// MinimumScratchBufferSize bytes. This may seem like a large value; however,
// when a program intends to enumerate large directories, having a larger
// scratch buffer results in fewer operating system calls.
const DefaultScratchBufferSize = 64 * 1024

// MinimumScratchBufferSize specifies the minimum size of the scratch buffer
// that Walk, ReadDirents, and ReadDirnames will use when reading file entries
// from the operating system. It is initialized to the result from calling
// `os.Getpagesize()` during program startup.
var MinimumScratchBufferSize int

func init() {
	MinimumScratchBufferSize = os.Getpagesize()
}

// Options provide parameters for how the Walk function operates.
type Options struct {
	// ErrorCallback specifies a function to be invoked in the case of an error
	// that could potentially be ignored while walking a file system
	// hierarchy.
	ErrorCallback func(string, error) ErrorAction

	// FollowSymbolicLinks specifies whether Walk will follow symbolic links
	// that refer to directories. When set to false or left as its zero-value,
	// Walk will still invoke the callback function with symbolic link nodes,
	// but if the symbolic link refers to a directory, it will not recurse on
	// that directory. When set to true, Walk will recurse on symbolic links
	// that refer to a directory.
	FollowSymbolicLinks bool

	// NoHidden (only UNIX) specifies whether Walk will follow hidden directories.
	// When set to false or left as its zero-value, Walk will invoke the callback
	// function and traverse into hidden directories too. When set to true, Walk
	// will not traverse to hidden directories.
	NoHidden bool

	// Ignore represents a list of names for directories that should be ignored.
	// When Walk is about to traverse a directory and directories name the same
	// to a string in the Ignore list then that direcory with all its descendants
	// is ignored. If is left empty as it is, it will be ignored by callback function.
	Ignore []string

	// Unsorted controls whether or not Walk will sort the immediate descendants
	// of a directory by their relative names prior to visiting each of those
	// entries.

	Unsorted bool

	// Callback is a required function that Walk will invoke for every file
	// system node it encounters.
	Callback WalkFunc

	// PostChildrenCallback is an option function that Walk will invoke for
	// every file system directory it encounters after its children have been
	// processed.
	PostChildrenCallback WalkFunc

	// ScratchBuffer is an optional byte slice to use as a scratch buffer for
	// Walk to use when reading directory entries, to reduce amount of garbage
	// generation.
	ScratchBuffer []byte
}

// ErrorAction defines a set of actions the Walk function could take based on
// the occurrence of an error while walking the file system. Halt or SkipNode
type ErrorAction int

const (
	Halt ErrorAction = iota
	SkipNode
)

// WalkFunc is the type of the function called for each file system node visited
// by Walk. The pathname argument will contain the argument to Walk as a prefix;
// that is, if Walk is called with "dir", which is a directory containing the
// file "a", the provided WalkFunc will be invoked with the argument "dir/a",
// using the correct os.PathSeparator for the Go Operating System architecture,
// GOOS. The directory entry argument is a pointer to a Dirent for the node,
// providing access to both the basename and the mode type of the file system
// node.
type WalkFunc func(osPathname string, directoryEntry *Dirent) error

// Walk walks the file tree rooted at the specified directory, calling the
// specified callback function for each file system node in the tree, including
// root, symbolic links, and other node types. The nodes are walked in lexical
// order, which makes the output deterministic but means that for very large
// directories this function can be inefficient.
func Walk(pathname string, options *Options) error {
	pathname = filepath.Clean(pathname)
	var fi os.FileInfo
	var err error
	if options.FollowSymbolicLinks {
		fi, err = os.Stat(pathname)
		if err != nil {
			return errors.Wrap(err, "cannot Stat")
		}
	} else {
		fi, err = os.Lstat(pathname)
		if err != nil {
			return errors.Wrap(err, "cannot Lstat")
		}
	}
	mode := fi.Mode()
	if mode&os.ModeDir == 0 {
		return errors.Errorf("cannot Walk non-directory: %s", pathname)
	}
	dirent := &Dirent{
		name:     filepath.Base(pathname),
		modeType: mode & os.ModeType,
	}
	if options.ErrorCallback == nil {
		options.ErrorCallback = func(_ string, _ error) ErrorAction { return Halt }
	}
	if len(options.ScratchBuffer) < MinimumScratchBufferSize {
		options.ScratchBuffer = make([]byte, DefaultScratchBufferSize)
	}
	err = walk(pathname, dirent, options)
	if err == filepath.SkipDir {
		return nil // silence SkipDir for top level
	}
	return err
}

// walk recursively traverses the file system node specified by pathname and the Dirent.
func walk(osPathname string, dirent *Dirent, options *Options) error {
	err := options.Callback(osPathname, dirent)
	if err != nil {
		if err == filepath.SkipDir {
			return err
		}
		err = errors.Wrap(err, "Callback")
		if action := options.ErrorCallback(osPathname, err); action == SkipNode {
			return nil
		}
		return err
	}

	if dirent.IsSymlink() {
		if !options.FollowSymbolicLinks {
			return nil
		}
		if !dirent.IsDir() {
			referent, err := os.Readlink(osPathname)
			if err != nil {
				err = errors.Wrap(err, "cannot Readlink")
				if action := options.ErrorCallback(osPathname, err); action == SkipNode {
					return nil
				}
				return err
			}
			var osp string
			if filepath.IsAbs(referent) {
				osp = referent
			} else {
				osp = filepath.Join(filepath.Dir(osPathname), referent)
			}
			fi, err := os.Stat(osp)
			if err != nil {
				err = errors.Wrap(err, "cannot Stat")
				if action := options.ErrorCallback(osp, err); action == SkipNode {
					return nil
				}
				return err
			}
			dirent.modeType = fi.Mode() & os.ModeType
		}
	}

	if !dirent.IsDir() {
		return nil
	}

	if options.NoHidden && string(dirent.name[0]) == "." {
		return nil
	}

	if len(options.Ignore) > 0 {
		for _, el := range options.Ignore {
			if dirent.name == el {
				return nil
			}
		}
	}
	deChildren, err := ReadDirents(osPathname, options.ScratchBuffer)
	if err != nil {
		err = errors.Wrap(err, "cannot ReadDirents")
		if action := options.ErrorCallback(osPathname, err); action == SkipNode {
			return nil
		}
		return err
	}

	if !options.Unsorted {
		sort.Sort(deChildren)
	}

	for _, deChild := range deChildren {
		osChildname := filepath.Join(osPathname, deChild.name)
		err = walk(osChildname, deChild, options)
		if err != nil {
			if err != filepath.SkipDir {
				return err
			}
			if deChild.IsSymlink() {
				if !deChild.IsDir() {
					referent, err := os.Readlink(osChildname)
					if err != nil {
						err = errors.Wrap(err, "cannot Readlink")
						if action := options.ErrorCallback(osChildname, err); action == SkipNode {
							continue
						}
						return err
					}
					var osp string
					if filepath.IsAbs(referent) {
						osp = referent
					} else {
						osp = filepath.Join(osPathname, referent)
					}
					fi, err := os.Stat(osp)
					if err != nil {
						err = errors.Wrap(err, "cannot Stat")
						if action := options.ErrorCallback(osp, err); action == SkipNode {
							continue // with next child
						}
						return err
					}
					deChild.modeType = fi.Mode() & os.ModeType
				}
			}
			if !deChild.IsDir() {
				return nil
			}
		}
	}
	if options.PostChildrenCallback == nil {
		return nil
	}
	err = options.PostChildrenCallback(osPathname, dirent)
	if err == nil || err == filepath.SkipDir {
		return err
	}
	err = errors.Wrap(err, "PostChildrenCallback") // wrap potential errors returned by callback
	if action := options.ErrorCallback(osPathname, err); action == SkipNode {
		return nil
	}
	return err
}

// ReadDirents returns a sortable slice of pointers to Dirent structures, each
// representing the file system name and mode type for one of the immediate
// descendant of the specified directory. If the specified directory is a
// symbolic link, it will be resolved.If an optional scratch buffer is provided
// that is at least one page of memory, it will be used when reading directory
//entries from the file system.
func ReadDirents(osDirname string, scratchBuffer []byte) (Dirents, error) {
	return readdirents(osDirname, scratchBuffer)
}

func readdirents(osDirname string, scratchBuffer []byte) (Dirents, error) {
	dh, err := os.Open(osDirname)
	if err != nil {
		return nil, errors.Wrap(err, "cannot Open")
	}

	var entries Dirents

	fd := int(dh.Fd())

	if len(scratchBuffer) < MinimumScratchBufferSize {
		scratchBuffer = make([]byte, DefaultScratchBufferSize)
	}

	var de *syscall.Dirent

	for {
		n, err := syscall.ReadDirent(fd, scratchBuffer)
		if err != nil {
			_ = dh.Close()
			return nil, errors.Wrap(err, "cannot ReadDirent")
		}
		if n <= 0 {
			break
		}
		buf := scratchBuffer[:n]
		for len(buf) > 0 {
			de = (*syscall.Dirent)(unsafe.Pointer(&buf[0]))
			buf = buf[de.Reclen:]

			if de.Ino == 0 {
				continue
			}

			nameSlice := func(de *syscall.Dirent) []byte {
				ml := int(uint64(de.Reclen) - uint64(unsafe.Offsetof(syscall.Dirent{}.Name)))
				var name []byte
				sh := (*reflect.SliceHeader)(unsafe.Pointer(&name))
				sh.Cap, sh.Len = ml, ml
				sh.Data = uintptr(unsafe.Pointer(&de.Name[0]))
				if index := bytes.IndexByte(name, 0); index >= 0 {
					sh.Cap = index
					sh.Len = index
				}
				return name
			}(de)

			namlen := len(nameSlice)
			if (namlen == 0) || (namlen == 1 && nameSlice[0] == '.') || (namlen == 2 && nameSlice[0] == '.' && nameSlice[1] == '.') {
				continue
			}
			osChildname := string(nameSlice)
			var mode os.FileMode
			switch de.Type {
			case syscall.DT_REG:
			case syscall.DT_DIR:
				mode = os.ModeDir
			case syscall.DT_LNK:
				mode = os.ModeSymlink
			case syscall.DT_CHR:
				mode = os.ModeDevice | os.ModeCharDevice
			case syscall.DT_BLK:
				mode = os.ModeDevice
			case syscall.DT_FIFO:
				mode = os.ModeNamedPipe
			case syscall.DT_SOCK:
				mode = os.ModeSocket
			default:
				fi, err := os.Lstat(filepath.Join(osDirname, osChildname))
				if err != nil {
					_ = dh.Close()
					return nil, errors.Wrap(err, "cannot Stat")
				}
				mode = fi.Mode() & os.ModeType
			}

			entries = append(entries, &Dirent{name: osChildname, modeType: mode})
		}
	}
	if err = dh.Close(); err != nil {
		return nil, err
	}
	return entries, nil
}

// ReadDirnames returns a slice of strings, representing the immediate
// descendants of the specified directory. If the specified directory is a
// symbolic link, it will be resolved. If an optional scratch buffer is provided
// that is at least one page of memory, it will be used when reading directory
// entries from the file system.
func ReadDirnames(osDirname string, scratchBuffer []byte) ([]string, error) {
	return readdirnames(osDirname, scratchBuffer)
}

func readdirnames(osDirname string, scratchBuffer []byte) ([]string, error) {
	des, err := readdirents(osDirname, scratchBuffer)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(des))
	for i, v := range des {
		names[i] = v.name
	}
	return names, nil
}
