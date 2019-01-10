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

func (dir File) ListDir(files Files) {
	list := chooseFile(IncFolder, IncFiles, IncHidden, Recurrent, dir)
	for _, d := range list {
		files = append(files, d)
	}
	return
}

func (dir File) Select(files Files, number int) (selected Files) {
	for i := range files {
		if files[i].Other.Selected {
			selected = append(selected, files[i])
		}
	}
	if len(selected) == 0 {
		selected = append(selected, files[number])
	}
	return
}

func (dir File) Copy(names Files) error {
	for _, file := range names {
		if err := cpAny(file.Path, dir.Path); err != nil {
			return fmt.Errorf("Could not copy file!")
		}
	}
	return nil
}

func (dir File) Move(names Files) error {
	for _, file := range names {
		if err := cpAny(file.Path, dir.Path); err != nil {
			return fmt.Errorf("Could not copy file")
		} else if err := os.RemoveAll(file.Path); err != nil {
			return fmt.Errorf("Could not delete file")
		}
	}
	return nil
}

func (dir File) Delete(names Files) error {
	for _, file := range names {
		if err := os.RemoveAll(file.Path); err != nil {
			return fmt.Errorf("Could not delete file")
		}
	}
	return nil
}

func (dir File) Write(name string, bytes []byte) error {
	newFileName := dir.Path + "/" + name
	newFileName = RenameExist(newFileName)
	if _, err := os.Create(newFileName); err != nil {
		return fmt.Errorf("Could not create file")
	} else if newFile, err := os.OpenFile(newFileName, os.O_RDWR|os.O_APPEND, 0777); err == nil {
		if _, err := newFile.Write(bytes); err != nil {
			return fmt.Errorf("Could not write file")
		} else {
			newFile.Close()
		}
	}
	return nil
}

func (dir File) Append(name string, bytes []byte) error {
	newFileName := dir.Path + "/" + name
	if _, err := os.Stat(newFileName); !os.IsNotExist(err) {
		if newFile, err := os.OpenFile(newFileName, os.O_RDWR|os.O_APPEND, 0777); err == nil {
			if _, err := newFile.Write(bytes); err != nil {
				return fmt.Errorf("Could not write file")
			} else {
				newFile.Close()
			}
		}
	} else {
		if err := dir.Write(name, bytes); err != nil {
			return err
		}
	}
	return nil
}

func (dir File) Overite(name string, bytes []byte) error {
	fileName := dir.Path + "/" + name
	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		if err := os.RemoveAll(fileName); err != nil {
			return fmt.Errorf("Could not delete file")
		}
	}
	if err := dir.Write(name, bytes); err != nil {
		return err
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

func (dir File) Rename(oldname, name string) error {
	newFileName := dir.Path + "/" + name
	oldFileName := dir.Path + "/" + oldname
	newFileName = RenameExist(newFileName)
	if err := os.Rename(oldFileName, newFileName); err != nil {
		return fmt.Errorf("Could not create folder")
	}
	return nil
}

func (dir File) Bulkname(names Files) error {
	for _, file := range names {
		print(file.Path)
	}
	return nil
}

type Explorer interface {
	ListDir(dir File) Files
	Select(files Files, number int) (selected Files)
	Move(path string, names []string) error
	Copy(path string, names []string) error
	Delete(path []string) error
	Rename(path, name string) error
	Bulkname(path []string) error
	Write(path, name string, bytes []byte) error
	Append(path, name string, bytes []byte) error
	Overite(path, name string, bytes []byte) error
	Mkdir(path, name string) error
	Run(path string) error
	RunWith(path string, app string) error
	Start(path string) error
	StartWith(path string, app string) error
	Edit(path string) error
}
