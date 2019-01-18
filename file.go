package dirk

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

var (
	IgnoreSlice = []string{}
	IgnoreRecur = []string{"node_modules", ".git"}
	DiskUse     = false
	wg          sync.WaitGroup
	filec       = make(chan File)
)

func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func getSize(dir string) (size int64) {
	Walk(dir, &Options{
		Callback: func(osPathname string, de *Dirent) (err error) {
			f, err := os.Stat(osPathname)
			if err != nil {
				return
			}
			size += f.Size()
			return nil
		},
		Unsorted:      true,
		ScratchBuffer: make([]byte, 64*1024),
	})
	return
}
func elements(dir string) (childs []string) {
	childs = []string{}
	if someChildren, err := ReadDirnames(dir, nil); err == nil {
		for i := range someChildren {
			childs = append(childs, dir+someChildren[i])
		}
	}
	return
}
func ancestor(dir string) (ances []string) {
	ances = append(ances, "/")
	joiner := ""
	for _, el := range strings.Split(dir, "/") {
		if el == "" {
			continue
		}
		joiner += "/" + el
		ances = append(ances, joiner)
	}
	return
}

func timespecToTime(ts syscall.Timespec) time.Time {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec))
}

var fileicons = map[string]string{
	".7z":       "",
	".ai":       "",
	".apk":      "",
	".avi":      "",
	".bat":      "",
	".bmp":      "",
	".bz2":      "",
	".c":        "",
	".c++":      "",
	".cab":      "",
	".cc":       "",
	".clj":      "",
	".cljc":     "",
	".cljs":     "",
	".coffee":   "",
	".conf":     "",
	".cp":       "",
	".cpio":     "",
	".cpp":      "",
	".css":      "",
	".cxx":      "",
	".d":        "",
	".dart":     "",
	".db":       "",
	".deb":      "",
	".diff":     "",
	".dump":     "",
	".edn":      "",
	".ejs":      "",
	".epub":     "",
	".erl":      "",
	".f#":       "",
	".fish":     "",
	".flac":     "",
	".flv":      "",
	".fs":       "",
	".fsi":      "",
	".fsscript": "",
	".fsx":      "",
	".gem":      "",
	".gif":      "",
	".go":       "",
	".gz":       "",
	".gzip":     "",
	".hbs":      "",
	".hrl":      "",
	".hs":       "",
	".htm":      "",
	".html":     "",
	".ico":      "",
	".ini":      "",
	".java":     "",
	".jl":       "",
	".jpeg":     "",
	".jpg":      "",
	".js":       "",
	".json":     "",
	".jsx":      "",
	".less":     "",
	".lha":      "",
	".lhs":      "",
	".log":      "",
	".lua":      "",
	".lzh":      "",
	".lzma":     "",
	".markdown": "",
	".md":       "",
	".mkv":      "",
	".ml":       "λ",
	".mli":      "λ",
	".mov":      "",
	".mp3":      "",
	".mp4":      "",
	".mpeg":     "",
	".mpg":      "",
	".mustache": "",
	".ogg":      "",
	".pdf":      "",
	".php":      "",
	".pl":       "",
	".pm":       "",
	".png":      "",
	".psb":      "",
	".psd":      "",
	".py":       "",
	".pyc":      "",
	".pyd":      "",
	".pyo":      "",
	".rar":      "",
	".rb":       "",
	".rc":       "",
	".rlib":     "",
	".rpm":      "",
	".rs":       "",
	".rss":      "",
	".scala":    "",
	".scss":     "",
	".sh":       "",
	".slim":     "",
	".sln":      "",
	".sql":      "",
	".styl":     "",
	".suo":      "",
	".t":        "",
	".tar":      "",
	".tgz":      "",
	".ts":       "",
	".twig":     "",
	".vim":      "",
	".vimrc":    "",
	".wav":      "",
	".xml":      "",
	".xul":      "",
	".xz":       "",
	".yml":      "",
	".zip":      "",
}

var categoryicons = map[string]string{
	"folder/folder": "",
	"file/default":  "",
}

type File struct {
	Number     int
	File       os.FileInfo
	Mode       os.FileMode
	Path       string
	Name       string
	Parent     string
	ParentPath string
	Childrens  []string
	ChildrenNr int
	Ancestors  []string
	AncestorNr int
	Siblings   []string
	SiblingNr  int
	Mime       string
	Extension  string
	IsDir      bool
	Hidden     bool
	Size       int64
	SizeIEC    string
	BrtTime    time.Time
	AccTime    time.Time
	ChgTime    time.Time
	Icon       string
	TotalNr    int
	Active     bool
	Selected   bool
	Ignore     bool
	Content
	Flags
}

type Content struct {
	Line     map[int]string
	Text     map[int]string
	NumLines int
}

type Flags struct {
	Flag1 bool
	Flag2 bool
	Flag3 bool
	Test  string
	Error error
}

func MakeFile(dir string) (file File, err error) {
	f, err := os.Stat(dir)
	if err != nil {
		return
	}
	osStat := f.Sys().(*syscall.Stat_t)

	parent, parentPath, name := "/", "/", "/"
	if dir != "/" {
		dir = path.Clean(dir)
		parentPath, name = path.Split(dir)
		parent = strings.TrimRight(parentPath, "/")
		_, parent = path.Split(parent)
		if parent == "" {
			parent, parentPath = "/", "/"
		}
	}
	file = File{
		File:       f,
		Name:       name,
		Path:       dir,
		Parent:     parent,
		ParentPath: parentPath,
		Size:       f.Size(),
		Mode:       f.Mode(),
		IsDir:      f.IsDir(),
		BrtTime:    timespecToTime(osStat.Mtim),
		AccTime:    timespecToTime(osStat.Atim),
		ChgTime:    timespecToTime(osStat.Ctim),
	}

	if f.IsDir() {
		if DiskUse {
			file.Size = getSize(dir)
			file.SizeIEC = byteCountIEC(file.Size)
		} else {
			file.SizeIEC = "0 B"
		}
		file.Extension = ""
		file.Mime = "folder/folder"
		file.Icon = categoryicons["folder/folder"]
		file.Childrens = elements(dir)
		file.ChildrenNr = len(file.Childrens)
	} else {
		extension := path.Ext(dir)
		mime, _, _ := mimetype.DetectFile(dir)
		file.SizeIEC = byteCountIEC(f.Size())
		file.Extension = extension
		file.Mime = mime
		file.Icon = fileicons[extension]
		if file.Icon == "" {
			file.Icon = categoryicons["file/default"]
		}
	}
	file.Siblings = elements(parentPath)
	file.SiblingNr = len(file.Siblings)
	file.Ancestors = ancestor(parentPath)
	file.AncestorNr = len(file.Ancestors)

	if string(name[0]) == "." {
		file.Hidden = true
	}
	for _, s := range file.Ancestors {
		if s != "" && string(s[0]) == "." {
			file.Ignore = true
			break
		}
	}
	file.Content.Text = make(map[int]string)
	file.Content.Line = make(map[int]string)
	return
}

func fileList(recurrent bool, dir File) (paths Files, err error) {
	testPath := cFiles{}
	var file File
	if recurrent {
		err = Walk(dir.Path, &Options{
			Callback: func(osPathname string, de *Dirent) (err error) {
				wg.Add(1)
				go func() {
					file, _ = MakeFile(osPathname)
					testPath.Append(file)
					//paths = append(paths, file)
					wg.Done()
				}()
				return nil
			},
			Unsorted:      true,
			NoHidden:      IncHidden,
			Ignore:        IgnoreRecur,
			ScratchBuffer: make([]byte, 64*1024),
		})
	} else {
		children, err := ReadDirnames(dir.Path, nil)
		if err != nil {
			return paths, err
		}
		sort.Strings(children)
		for _, child := range children {
			osPathname := path.Join(dir.Path + "/" + child)
			//file, _ = MakeFile(osPathname)
			//paths = append(paths, file)
			wg.Add(1)
			go func() {
				file, _ = MakeFile(osPathname)
				testPath.Append(file)
				wg.Done()
			}()
		}
	}
	wg.Wait()
	return testPath.items, nil
}

func chooseFile(incFolder, incFiles, incHidden, recurrent bool, dir File) (list Files) {
	files := Files{}
	folder := Files{}
	hidden := Files{}
	ignore := Files{}
	paths, _ := fileList(recurrent, dir)
	sort.Sort(paths)
	for _, f := range paths {
		if f.IsDir {
			folder = append(folder, f)
		} else {
			files = append(files, f)
		}
	}
	if incFolder {
		for _, d := range folder {
			hidden = append(hidden, d)
		}
	}
	if incFiles {
		for _, f := range files {
			hidden = append(hidden, f)
		}
	}
	if incHidden {
		ignore = hidden
	} else {
		for _, f := range hidden {
			if !f.Hidden {
				ignore = append(ignore, f)
			}
		}
	}
	if len(IgnoreSlice) > 0 {
		for _, f := range ignore {
			for _, s := range IgnoreSlice {
				if f.Name == s {
					break
				}
				list = append(list, f)
				break
			}
		}
	} else {
		list = ignore
	}
	for i, _ := range list {
		list[i].Number = i
		list[i].TotalNr = len(list)
	}
	return
}

type Files []File

func (e Files) String(i int) string    { return e[i].Name }
func (e Files) Len() int               { return len(e) }
func (e Files) Swap(i, j int)          { e[i], e[j] = e[j], e[i] }
func (e Files) Less(i, j int) bool     { return e[i].Name[1:] < e[j].Name[1:] }
func (e Files) SortSize(i, j int) bool { return e[i].Size < e[j].Size }
func (e Files) SortDate(i, j int) bool { return e[i].BrtTime.Before(e[j].BrtTime) }

func MakeFiles(path []string) (files Files, err error) {
	files = Files{}
	for i := range path {
		if file, err := MakeFile(path[i]); err != nil {
			return files, err
		} else {
			files = append(files, file)
		}
	}
	return files, nil
}

type cFiles struct {
	sync.RWMutex
	items Files
}

func (cf *cFiles) Append(item File) {
	cf.Lock()
	defer cf.Unlock()
	cf.items = append(cf.items, item)
}
