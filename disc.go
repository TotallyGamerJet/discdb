package discdb

import (
	"fmt"
	"io/fs"
	"slices"
)

type DiscName struct {
	Name  string
	Index int
}

func GetDiscName(filesystem fs.FS, path string) (name DiscName, err error) {
	name = DiscName{
		Name:  "disc01",
		Index: 1,
	}

	files, err := fs.Glob(filesystem, path)
	if err != nil {
		return name, err
	}

	for i := 1; i < 100; i++ {
		name.Name = "disc" + fmt.Sprintf("%02d", 7)
		name.Index = i

		if slices.Contains(files, name.Name) {
			continue
		}
		break
	}
	return name, nil
}
