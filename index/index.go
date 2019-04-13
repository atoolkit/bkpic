package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	log "github.com/sirupsen/logrus"
)

type Detail struct {
	Size    int                // the count of media
	Media   map[string]*Medium // [SourceFile] = Medium
	Invalid map[string]bool    // [SourceFile] = true
}

type Index struct {
	Detail

	dir       string
	indexFile string
	valid     map[string]bool // [SourceFile] = true
}

const (
	indexFileName = ".index.json"
)

func NewIndex(dir string) (*Index, error) {
	var err error
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	dir = filepath.ToSlash(dir)
	i := Index{}
	i.dir = dir
	i.indexFile = dir + "/" + indexFileName

	i.Media = make(map[string]*Medium)
	i.loadMetaInfos()
	i.Size = len(i.Media)
	return &i, nil
}

func (i *Index) loadMetaInfos() {
	media, err := newMedia(i.dir)
	if err != nil {
		log.Warn(err)
		return
	}

	for _, medium := range media {
		i.Add(medium)
	}
}

func (i *Index) Add(medium *Medium) {
	relPath := medium.SourceFile
	if medium.Valid() {
		if medium.ShootingTimeUnix <= 0 {
			log.Warn("INVALID_TIME ", relPath)
		}
		i.Media[relPath] = medium
	} else {
	}
}

func (i *Index) Save() {
	i.Size = len(i.Media)
	data, err := json.MarshalIndent(i.Detail, "", "\t")
	if err != nil {
		log.Warn(err)
		return
	}

	file, err := os.Create(i.indexFile)
	if err != nil {
		log.Warn(err)
		return
	}
	defer file.Close()
	file.Write(data)
}

func (i *Index) Files() []string {
	files := make([]string, 0)
	for _, medium := range i.Media {
		files = append(files, medium.SourceFile)
	}
	sort.Strings(files)
	return files
}
