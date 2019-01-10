package dirk

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

var (
	IncFolder = true
	IncFiles  = true
	IncHidden = false
	Recurrent = false
	cmd       *exec.Cmd
)

/*
	Open a file, directory, or URI using the OS's default
	application for that object type. Wait for the open
	command to complete.
*/
func Run(input string) error {
	return open(input).Run()
}

/*
	Open a file, directory, or URI using the OS's default
	application for that object type. Don't wait for the
	open command to complete.
*/
func Start(input string) error {
	return open(input).Start()
}

/*
	Open a file, directory, or URI using the specified application.
	Wait for the open command to complete.
*/
func RunWith(input string, appName string) error {
	return openWith(input, appName).Run()
}

/*
	Open a file, directory, or URI using the specified application.
	Don't wait for the open command to complete.
*/
func StartWith(input string, appName string) error {
	return openWith(input, appName).Start()
}

func open(input string) *exec.Cmd {
	return exec.Command("xdg-open", input)
}

func openWith(input string, appName string) *exec.Cmd {
	return exec.Command(appName, input)
}

func Edit(file string) error {
	editor := os.Getenv("EDITOR")
	if len(editor) > 0 {
		cmd = exec.Command(editor, file)
	} else {
		cmd = exec.Command("/usr/bin/env", "nvim", file)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Error:", err)
	}
	return nil
}

func RenameExist(name string) string {
	if _, err := os.Stat(name); err == nil {
		i := 1
		for {
			if _, err := os.Stat(name + strconv.Itoa(i)); err == nil {
				i++
			} else {
				break
			}
		}
		return name + strconv.Itoa(i)
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

func ListDir(dir File) (files Files) {
	list := chooseFile(IncFolder, IncFiles, IncHidden, Recurrent, dir)
	for _, d := range list {
		files = append(files, d)
	}
	return
}

func Select(files Files, number int) []string {
	selected := []string{}
	for i := range files {
		if files[i].Other.Selected {
			selected = append(selected, files[i].Path)
		}
	}
	if len(selected) < 1 {
		selected = append(selected, files[number].Path)
	}
	return selected
}

func Copy(path []string, newPath string) error {
	for _, file := range path {
		if err := cpAny(file, newPath); err != nil {
			return fmt.Errorf("Could not copy file!")
		}
	}
	return nil
}

func Move(path []string, newPath string) error {
	for _, file := range path {
		if err := cpAny(file, newPath); err != nil {
			return fmt.Errorf("Could not copy file")
		} else if err := os.RemoveAll(file); err != nil {
			return fmt.Errorf("Could not delete file")
		}
	}
	return nil
}

func Delete(path []string, newPath string) error {
	for _, file := range path {
		if err := os.RemoveAll(file); err != nil {
			return fmt.Errorf("Could not delete file")
		}
	}
	return nil
}

func Touch(path, name string) error {
	name = RenameExist(name)
	if newFile, err := os.Create(name); err != nil {
		return fmt.Errorf("Could not create file")
	} else {
		newFile.Close()
	}
	return nil
}

func Mkdir(path, name string) error {
	name = RenameExist(name)
	if err := os.MkdirAll(name, 0777); err != nil {
		return fmt.Errorf("Could not create folder")
	}
	return nil
}

type Explorer interface {
	ListDir(dir File) Files
	Select(files Files, number int) []string
	Move(path []string, newPath string) error
	Copy(path []string, newPath string) error
	Delete(path []string) error
	Touch(path, name string) error
	Mkdir(path, name string) error
	Run(path string) error
	RunWith(path string, app string) error
	Start(path string) error
	StartWith(path string, app string) error
	Edit(path string) error
}
