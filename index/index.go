package index

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

type Index struct {
	mediaBySize map[int64]Media
	media       map[string]*Medium
	dir         string
}

func NewEmptyIndex() *Index {
	return &Index{
		mediaBySize: make(map[int64]Media),
		media:       make(map[string]*Medium),
	}
}

func NewIndex(dir string) (*Index, error) {
	idx := NewEmptyIndex()
	if err := idx.Walk(dir); err != nil {
		return nil, err
	}
	idx.dir = dir
	return idx, nil
}

func (idx *Index) Walk(dir string) error {
	var err error
	dir, err = filepath.Abs(dir)
	if err != nil {
		return err
	}

	if err := filepath.Walk(dir, idx.walk); err != nil {
		return err
	}
	return nil
}

func (idx *Index) walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() || info.Size() <= 0 {
		return nil
	}

	idx.Add(path)
	return nil
}

func (idx *Index) Add(fullPath string) {
	medium := NewMedium(fullPath)
	if medium == nil {
		//zap.L().Error("invalid medium", zap.String("file", fullPath))
		return
	}

	info := medium.FileInfo

	hashes, ok := idx.mediaBySize[info.Size()]
	if !ok {
		hashes = make(Media, 0)
	}

	hashes = append(hashes, medium)
	idx.mediaBySize[info.Size()] = hashes
	idx.media[medium.FullPath] = medium
}

func (idx *Index) Directory() string {
	return idx.dir
}

func (idx *Index) Get(fullPath string) *Medium {
	return idx.media[fullPath]
}

func (idx *Index) GetMediaBySize() map[int64]Media {
	return idx.mediaBySize
}

func (idx *Index) Same(medium *Medium) *Medium {
	media, ok := idx.mediaBySize[medium.FileInfo.Size()]
	if !ok {
		return nil
	}

	return media.Same(medium)
}

func (idx *Index) Size() int {
	return len(idx.media)
}

func (idx *Index) LoadMeta() error {

	args := append(exiftoolFlags, idx.dir)
	cmd := exec.Command("exiftool", args...)
	zap.L().Debug(cmd.String())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	decoder := json.NewDecoder(stdout)
	var meta []*Meta
	if err := decoder.Decode(&meta); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	if len(meta) <= 0 {
		zap.L().Info("no media", zap.String("directory", idx.dir))
		return nil
	}

	for _, m := range meta {
		medium, ok := idx.media[m.SourceFile]
		if !ok {
			continue
		}
		medium.meta = m
	}
	return nil
}
