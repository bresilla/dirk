//source https://github.com/rodkranz/ff/
package dirk

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

type Finder struct {
	Text  string
	Regex *regexp.Regexp
	list  Files
}

func (f *Finder) searchByText(elem *File, numLine int, line string) {
	if f.Regex != nil {
		return
	}
	if strings.Contains(line, f.Text) {
		elem.Flags.Flag1 = true
		elem.Content.Line[numLine] = line
		elem.Content.Text[numLine] = f.Text
	}
}

func (f *Finder) searchByRegex(elem *File, numLine int, line string) {
	if f.Regex == nil {
		return
	}
	words := f.Regex.FindAllString(line, -1)
	if len(words) > 0 {
		for _, v := range words {
			elem.Flags.Flag1 = true
			elem.Content.Line[numLine] = line
			elem.Content.Text[numLine] = v
		}
	}
}

func (f *Finder) readAndFind(e *File) {
	file, err := os.Open(e.Path)
	if err != nil {
		return
	}
	defer file.Close()
	f.Text = strings.ToLower(f.Text)
	numLine := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		numLine++
		line = strings.ToLower(line)
		f.searchByText(e, numLine, line)
		f.searchByRegex(e, numLine, line)
	}
	e.Content.NumLines = numLine
}

func (f *Finder) FindText(text string) (files []File) {
	if len(f.Text) == 0 && f.Regex == nil {
		return f.list
	}
	f.Text = text
	for i, element := range f.list {
		f.readAndFind(&element)
		f.list[i] = element
	}
	for i := range f.list {
		if f.list[i].Flags.Flag1 {
			files = append(files, f.list[i])
		}
	}
	return
}
