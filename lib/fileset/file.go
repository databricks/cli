package fileset

import (
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

type File struct {
	fs.DirEntry
	Absolute string
	Relative string
}

func (f File) Modified() (ts time.Time) {
	info, err := f.Info()
	if err != nil {
		// return default time, beginning of epoch
		return ts
	}
	return info.ModTime()
}

func (fi File) Ext(suffix string) bool {
	return strings.HasSuffix(fi.Name(), suffix)
}

func (fi File) Dir() string {
	return path.Dir(fi.Absolute)
}

func (fi File) MustMatch(needle string) bool {
	return fi.Match(regexp.MustCompile(needle))
}

func (fi File) FindAll(needle *regexp.Regexp) (all []string, err error) {
	raw, err := fi.Raw()
	if err != nil {
		log.Printf("[ERROR] read %s: %s", fi.Absolute, err)
		return nil, err
	}
	for _, v := range needle.FindAllStringSubmatch(string(raw), -1) {
		all = append(all, v[1])
	}
	return all, nil
}

func (fi File) Match(needle *regexp.Regexp) bool {
	raw, err := fi.Raw()
	if err != nil {
		log.Printf("[ERROR] read %s: %s", fi.Absolute, err)
		return false
	}
	return needle.Match(raw)
}

func (fi File) Raw() ([]byte, error) {
	f, err := fi.Open()
	if err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}

func (fi File) Open() (*os.File, error) {
	return os.Open(fi.Absolute)
}

