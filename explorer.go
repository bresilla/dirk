package dirk

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	t "github.com/bresilla/shko/term"
)

var (
	IncFolder   = true
	IncFiles    = true
	IncHidden   = false
	Recurrent   = false
	DiskUse     = false
	IgnoreSlice = []string{".git"}
	IgnoreRecur = []string{"node_modules", ".git"}
)

func open(input string) *exec.Cmd {
	return exec.Command("xdg-open", input)
}

func openWith(input string, appName string) *exec.Cmd {
	return exec.Command(appName, input)
}

func renameExist(name string) string {
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
	dst = renameExist(dst)
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
	dst = renameExist(dst)
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
			dst = renameExist(dst)
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

func unzip(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}
		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()
		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()
		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
	}
	return nil
}

func zipit(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func tarit(source, target string) error {
	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.tar", filename))
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if err := tarball.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			return err
		})
}

func untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return err
}

func ungzip(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	target = filepath.Join(target, archive.Name)
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return err
}

func gzipit(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}

	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.gz", filename))
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	archiver := gzip.NewWriter(writer)
	archiver.Name = filename
	defer archiver.Close()

	_, err = io.Copy(archiver, reader)
	return err
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

func unique(intSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func (dir File) ListDir() Files {
	files := Files{}
	list := chooseFile(IncFolder, IncFiles, IncHidden, Recurrent, dir)
	for _, d := range list {
		files = append(files, d)
	}
	return files
}

func (dir File) Select(files Files) Files {
	selected := Files{}
	for i := range files {
		if files[i].Selected || files[i].Active {
			selected = append(selected, files[i])
		}
	}
	return selected
}

func (dir File) Touch(names ...string) (Files, error) {
	files := Files{}
	for i := range names {
		newFileName := dir.Path + "/" + names[i]
		newFileName = renameExist(newFileName)
		if newFile, err := os.Create(newFileName); err != nil {
			return files, fmt.Errorf("Could not create file")
		} else {
			theFile, _ := MakeFile(newFileName)
			files = append(files, &theFile)
			newFile.Close()
		}
	}
	return files, nil
}

func (dir File) Mkdir(names ...string) (Files, error) {
	files := Files{}
	for i := range names {
		newFileName := dir.Path + "/" + names[i]
		newFileName = renameExist(newFileName)
		if err := os.MkdirAll(newFileName, 0777); err != nil {
			return files, fmt.Errorf("Could not create folder")
		} else {
			theFile, _ := MakeFile(newFileName)
			files = append(files, &theFile)
		}
	}
	return files, nil
}

func (files Files) Paste(destin File) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if _, err := os.Stat(files[i].Path); !os.IsNotExist(err) {
			if err := cpAny(files[i].Path, destin.Path); err != nil {
				return fmt.Errorf("Could not copy file!")
			}
		}
	}
	return nil
}

func (files Files) Move(destin File) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		if _, err := os.Stat(files[i].Path); !os.IsNotExist(err) {
			if err := cpAny(files[i].Path, destin.Path); err != nil {
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

func (files Files) Read2B() ([][]byte, error) {
	fileArray := [][]byte{}
	if len(files) == 0 {
		return fileArray, fmt.Errorf("No file selected")
	}
	for i := range files {
		jointMem, err := ioutil.ReadFile(files[i].Path)
		if err != nil {
			return fileArray, err
		}
		fileArray = append(fileArray, jointMem)
	}
	return fileArray, nil
}

func (files Files) Read2S() ([][]string, error) {
	fileArray := [][]string{}
	if len(files) == 0 {
		return fileArray, fmt.Errorf("No file selected")
	}
	for i := range files {
		file, err := os.Open(files[i].Path)
		if err != nil {
			return fileArray, err
		}
		defer file.Close()
		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		fileArray = append(fileArray, lines)
	}
	return fileArray, nil
}

func (files Files) Write(bytes []byte) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		newFileName := renameExist(files[i].Path)
		if _, err := os.Create(newFileName); err != nil {
			return fmt.Errorf("Could not create file")
		} else if newFile, err := os.OpenFile(newFileName, os.O_RDWR|os.O_APPEND, 0777); err == nil {
			if _, err := newFile.Write(bytes); err != nil {
				return fmt.Errorf("Could not write file")
			}
			newFile.Close()
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
				}
				newFile.Close()
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

func (files Files) Union(name string) error {
	isMixed := false
	if len(files) < 1 {
		return fmt.Errorf("Not enough files to join")
	}
	for i := range files {
		if files[i].IsDir {
			isMixed = true
		}
	}
	virtDir, _ := MakeFile(files[0].ParentPath)
	if !isMixed {
		toWrite, _ := virtDir.Touch(name)
		for i := range files {
			toWrite.Append([]byte(strconv.Itoa(i)))
		}
		t.PrintWait("file")
	} else {
		toPlace, _ := virtDir.Mkdir(name)
		files.Paste(*toPlace[0])
	}
	return nil
}

func (files Files) Indent(name string) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	virtDir, _ := MakeFile(files[0].ParentPath)
	toPlace, _ := virtDir.Mkdir(name)
	files.Paste(*toPlace[0])
	files.Delete()
	return nil
}

func (files Files) Rename(name ...string) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	parent := files[0].ParentPath
	if len(files) == len(name) {
		for i := range files {
			newFileName := renameExist(parent + "/" + name[i])
			if err := os.Rename(files[i].Path, newFileName); err != nil {
				return fmt.Errorf("Could not create folder")
			}
		}
	} else {
		if len(files) > 1 {
			parentDir, _ := MakeFile(parent)
			tempFile, _ := parentDir.Touch(".temp")
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
				newName := renameExist(name)
				os.Rename(files[i].Path, files[i].ParentPath+newName)
			}
			tempFile.Delete()
		}
	}
	return nil
}

func (files Files) Archive(name string) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	archSlice := []string{}
	for i := range files {
		archSlice = append(archSlice, files[i].Path)
	}
	for i := range files {
		extension := filepath.Ext(name)
		newFileName := renameExist(files[i].ParentPath + "/" + name)
		switch extension {
		case ".zip":
			zipit(files[i].Path, newFileName)
		case ".tar":
			tarit(files[i].Path, newFileName)
		case ".tgz", ".tar.gz":
			gzipit(files[i].Path, newFileName)
		default:
			return fmt.Errorf("Not a proper archive name")
		}
	}
	return nil
}

func (files Files) Unarchive(name string) error {
	if len(files) == 0 {
		return fmt.Errorf("No file selected")
	}
	for i := range files {
		newFileName := renameExist(files[i].ParentPath + "/" + name)
		switch files[i].Exte {
		case ".zip":
			unzip(files[i].Path, newFileName)
		case ".tar":
			untar(files[i].Path, newFileName)
		case ".tgz", ".tar.gz":
			ungzip(files[i].Path, newFileName)
		default:
			return fmt.Errorf("Not an archive")
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
	var cmd *exec.Cmd
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

func (files Files) Match(finder Finder) Files {
	matched := Files{}
	if finder.Regex == nil {
		result := FindFrom(finder.Text, files)
		for _, r := range result {
			matched = append(matched, files[r.Index])
		}
	} else {
	}
	return matched
}

func (files Files) Find(finder Finder) Files {
	matched := Files{}
	if len(finder.Text) == 0 && finder.Regex == nil {
		return files
	}
	for i := range files {
		if files[i].Mime[:4] != "text" {
			continue
		}
		readAndFind(files[i], finder)
	}
	for i := range files {
		if len(files[i].MapLine) > 0 {
			matched = append(matched, files[i])
		}
	}
	return matched
}
