package dirk

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/mholt/archiver"
)

var (
	IncFolder = true
	IncFiles  = true
	IncHidden = false
	Recurrent = false
	cmd       *exec.Cmd
)

func open(input string) *exec.Cmd {
	return exec.Command("xdg-open", input)
}

func openWith(input string, appName string) *exec.Cmd {
	return exec.Command(appName, input)
}

func RenameExist(name string) string {
	if _, err := os.Stat(name); err == nil {
		i := 1
		for {
			if _, err := os.Stat(name + "(" + strconv.Itoa(i) + ")"); err == nil {
				i++
			} else {
				break
			}
		}
		return name + "(" + strconv.Itoa(i) + ")"
	}
	return name
}

func cpFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	var out *os.File
	defer in.Close()
	dst = RenameExist(dst)
	out, err = os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	err = out.Sync()
	if err != nil {
		return err
	}
	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, si.Mode())
}

func cpDir(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}
	dst = RenameExist(dst)
	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}
	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			err = cpDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}
			err = cpFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}
	return err
}

func cpAny(src, dst string) error {
	srcinfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if srcinfo.IsDir() {
		dstinfo, err := os.Stat(dst)
		if err == nil {
			if os.SameFile(srcinfo, dstinfo) {
				return fmt.Errorf("directory is itself: %s", dst)
			}
			dst += "/" + filepath.Base(src)
			dst = RenameExist(dst)
			return cpDir(src, dst)
		}
		return cpDir(src, dst)
	}
	dstinfo, err := os.Stat(dst)
	if err == nil {
		if dstinfo.IsDir() {
			return cpFile(src, dst+"/"+filepath.Base(src))
		}
		if os.SameFile(srcinfo, dstinfo) {
			return nil
		}
		return cpFile(src, dst)
	}
	return cpFile(src, dst)
}

func createDir(dirName string) bool {
	src, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dirName, 0755)
		if errDir != nil {
			panic(err)
		}
		return true
	}
	if src.Mode().IsRegular() {
		return false
	}
	return false
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func (dir File) ListDir() Files {
	files := Files{}
	list := chooseFile(IncFolder, IncFiles, IncHidden, Recurrent, dir)
	for _, d := range list {
		files = append(files, d)
	}
	return files
}

func (dir File) Select(files Files, number int) Files {
	selected := Files{}
	for i := range files {
		if files[i].Selected {
			selected = append(selected, files[i])
		}
	}
	if len(selected) == 0 {
		selected = append(selected, files[number])
	}
	return selected
}

func (dir File) Touch(name string) error {
	newFileName := dir.Path + "/" + name
	newFileName = RenameExist(newFileName)
	if newFile, err := os.Create(newFileName); err != nil {
		return fmt.Errorf("Could not create file")
	} else {
		newFile.Close()
	}
	return nil
}

func (dir File) Mkdir(name string) error {
	newFileName := dir.Path + "/" + name
	newFileName = RenameExist(newFileName)
	if err := os.MkdirAll(newFileName, 0777); err != nil {
		return fmt.Errorf("Could not create folder")
	}
	return nil
}

func (files Files) Paste(dir File) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if _, err := os.Stat(files[i].Path); !os.IsNotExist(err) {
			if err := cpAny(files[i].Path, dir.Path); err != nil {
				return fmt.Errorf("Could not copy file!")
			}
		}
	}
	return nil
}

func (files Files) Move(dir File) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if _, err := os.Stat(files[i].Path); !os.IsNotExist(err) {
			if err := cpAny(files[i].Path, dir.Path); err != nil {
				return fmt.Errorf("Could not copy file")
			} else if err := os.RemoveAll(files[i].Path); err != nil {
				return fmt.Errorf("Could not delete file")
			}
		}
	}
	return nil
}

func (files Files) Delete() error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if err := os.RemoveAll(files[i].Path); err != nil {
			return fmt.Errorf("Could not delete file")
		}
	}
	return nil
}

func (files Files) Write(bytes []byte) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		newFileName := RenameExist(files[i].Path)
		if _, err := os.Create(newFileName); err != nil {
			return fmt.Errorf("Could not create file")
		} else if newFile, err := os.OpenFile(newFileName, os.O_RDWR|os.O_APPEND, 0777); err == nil {
			if _, err := newFile.Write(bytes); err != nil {
				return fmt.Errorf("Could not write file")
			} else {
				newFile.Close()
			}
		}
	}
	return nil
}

func (files Files) Append(bytes []byte) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if _, err := os.Stat(files[i].Path); !os.IsNotExist(err) {
			if newFile, err := os.OpenFile(files[i].Path, os.O_RDWR|os.O_APPEND, 0777); err == nil {
				if _, err := newFile.Write(bytes); err != nil {
					return fmt.Errorf("Could not write file")
				} else {
					newFile.Close()
				}
			}
		} else {
			if err := files.Write(bytes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (files Files) Overite(bytes []byte) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	if err := files.Delete(); err != nil {
		return err
	}
	if err := files.Write(bytes); err != nil {
		return err
	}
	return nil
}

func (files Files) Rename(name ...string) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	parent := files[0].ParentPath
	if len(files) == len(name) {
		for i := range files {
			newFileName := RenameExist(parent + "/" + name[i])
			if err := os.Rename(files[i].Path, newFileName); err != nil {
				return fmt.Errorf("Could not create folder")
			}
		}
	} else {
		if len(files) > 1 {
			parentDir, _ := MakeFile(parent)
			parentDir.Touch(".temp")
			tempFile, _ := MakeFiles([]string{parentDir.Path + "/.temp"})
			for i := range files {
				tempFile.Append([]byte(files[i].Name + "\n"))
			}
			if err := tempFile.Edit(); err != nil {
				return err
			}
			fmt.Print("\033[?25l")
			newNames, _ := readLines(tempFile[0].Path)
			if len(newNames) != len(files) {
				tempFile.Delete()
				return fmt.Errorf("Number of files and names don't match")
			}
			for i, name := range newNames {
				os.Rename(files[i].Path, files[i].ParentPath+name)
			}
			tempFile.Delete()
		}
	}
	return nil
}

func (files Files) Archive(extension string) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	archSlice := []string{}
	for i := range files {
		archSlice = append(archSlice, files[i].Path)
	}
	newFileName := RenameExist(files[0].Parent + "." + extension)
	if err := archiver.Archive(archSlice, newFileName); err != nil {
		return err
	}
	return nil
}

func (files Files) Unarchive() error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if err := archiver.Unarchive(files[i].Path, files[i].Path+"_E"); err != nil {
			continue
		}
	}
	return nil
}

func (files Files) Run() error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if err := open(files[i].Path).Run(); err != nil {
			return fmt.Errorf("Could not open file")
		}
	}
	return nil
}

func (files Files) Start() error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if err := open(files[i].Path).Start(); err != nil {
			return fmt.Errorf("Could not open file")
		}
	}
	return nil
}

func (files Files) RunWith(name string) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if err := openWith(name, files[i].Path).Run(); err != nil {
			return fmt.Errorf("Could no open file")
		}
	}
	return nil
}

func (files Files) StartWith(name string) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if err := openWith(name, files[i].Path).Start(); err != nil {
			return fmt.Errorf("Could not open file")
		}
	}
	return nil
}

func (files Files) Edit() error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		editor := os.Getenv("EDITOR")
		if len(editor) > 0 {
			cmd = exec.Command(editor, files[i].Path)
		} else {
			cmd = exec.Command("/usr/bin/env", "nvim", files[i].Path)
		}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("Error:", err)
		}
	}
	return nil
}

func (files Files) Match(pattern string) Files {
	matched := Files{}
	result := FindFrom(pattern, files)
	for _, r := range result {
		matched = append(matched, files[r.Index])
	}
	return matched
}

func (files Files) Find(reg *regexp.Regexp, text string) Files {
	newFinder := Finder{
		list:  files,
		Regex: reg,
		Text:  text,
	}
	return newFinder.FindText(text)
}

type Explorer interface {
	ListDir() Files
	Select(files Files, number int) Files
	Match(pattern string) Files
	Find(reg *regexp.Regexp, text string) Files
	Touch(name string) error
	Mkdir(name string) error

	Move(dir File) error
	Paste(dir File) error
	Delete() error
	Write(bytes []byte) error
	Append(bytes []byte) error
	Overite(bytes []byte) error
	Rename(name []string) error
	Run() error
	RunWith(name string) error
	Start() error
	StartWith(name string) error
	Edit() error
}
