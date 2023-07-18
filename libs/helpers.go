package libs

import "os"

func ChdirAndBack(dir string) func() {
	wd, _ := os.Getwd()
	os.Chdir(dir)
	return func() {
		os.Chdir(wd)
	}
}
