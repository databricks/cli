package folders

import (
	"errors"
	"fmt"
	"os"
	"path"
)

func FindDirWithLeaf(leaf string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot find $PWD: %s", err)
	}
	for {
		_, err = os.Stat(fmt.Sprintf("%s/%s", dir, leaf))
		if errors.Is(err, os.ErrNotExist) {
			// TODO: test on windows
			next := path.Dir(dir)
			if dir == next { // or stop at $HOME?..
				return "", fmt.Errorf("cannot find %s anywhere", leaf)
			}
			dir = next
			continue
		}
		if err != nil {
			return "", err
		}
		return dir, nil
	}
}
