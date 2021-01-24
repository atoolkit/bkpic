package index

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

const (
	blockSize int64 = 4096
)

var (
	exiftoolFlags = []string{"-a", "-charset", "FileName=UTF8", "-d", "%s", "-ee", "--ext", "json", "-G", "-j", "-L", "-q", "-r", "-sort"}
)

func newMedia(dir string) ([]*Medium, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	args := append(exiftoolFlags, absDir)
	cmd := exec.Command("exiftool", args...)
	zap.S().Info(args)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		zap.S().Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		zap.S().Fatal(err)
	}

	decoder := json.NewDecoder(stdout)
	var media []*Medium
	if err := decoder.Decode(&media); err != nil {
		zap.S().Errorf("error when %d", len(media))
		if len(media) > 0 {
			zap.S().Errorf("%s", media[len(media)-1].SourceFile)
		}
	}

	if err := cmd.Wait(); err != nil {
		zap.S().Warn(err)
	}

	if len(media) <= 0 {
		return nil, fmt.Errorf("%s has no media", absDir)
	}

	for _, m := range media {
		m.init(absDir)
	}
	return media, nil
}
