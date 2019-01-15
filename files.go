package dirk

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
