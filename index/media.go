package index

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	blockSize int64 = 4096
)

var (
	exiftoolFlags = []string{"-a", "-d", "%s", "-ee", "--ext", "json", "-fast2", "-G", "-j", "-L", "-q", "-r", "-sort"}
)

func newMedia(dir string) ([]*Medium, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	exifPath := absDir + "/.exif.json"
	if _, err := os.Stat(exifPath); err == nil {
		err := os.Remove(exifPath)
		if err != nil {
			return nil, err
		}
	}
	// exifFile, err := os.Open(exifPath)
	// if err != nil {
	exifFile, err := os.Create(absDir + "/" + ".exif.json")
	if err != nil {
		return nil, err
	}
	defer exifFile.Close()

	args := append(exiftoolFlags, absDir)
	cmd := exec.Command("exiftool", args...)
	cmd.Stdout = exifFile

	err = cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			err = errors.New(bytes.NewBuffer(ee.Stderr).String())
		}
		return nil, err
	}
	//exifFile = exifOut
	// }

	exifFile.Seek(0, 0)
	stat, err := exifFile.Stat()
	if err != nil {
		return nil, err
	}

	b := make([]byte, stat.Size())
	_, err = exifFile.Read(b)
	if err != nil {
		return nil, err
	}

	exifFile.Seek(0, 0)
	var d *json.Decoder
	if utf8.Valid(b) {
		d = json.NewDecoder(exifFile)
	} else {
		r := transform.NewReader(exifFile, simplifiedchinese.GB18030.NewDecoder())
		d = json.NewDecoder(r)
	}

	var media []*Medium
	if err := d.Decode(&media); err != nil {
		log.Errorf("error when %d", len(media))
		return nil, err
	}

	if len(media) <= 0 {
		return nil, fmt.Errorf("%s has no media", absDir)
	}

	if err != nil {
		log.Warn(err, " ", media[len(media)-1].SourceFile)
		media = media[:len(media)-1]
	}

	for _, m := range media {
		m.init(absDir)
	}
	return media, nil
}
