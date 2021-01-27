package index

import (
	"crypto/sha256"
	"encoding/json"
	"hash/adler32"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/corona10/goimagehash"
	"go.uber.org/zap"
)

const (
	sizeForAdler32 = 4096
	imagePrefix    = "image/"
	videoPrefix    = "video/"
)

var (
	exiftoolFlags = []string{"-a", "-charset", "FileName=UTF8", "-d", "%s", "-ee", "--ext", "json", "-G", "-j", "-L", "-q", "-r", "-sort"}
	validDataTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
)

type Meta struct {
	Directory      string `json:"File:Directory"`
	FileModifyDate int64  `json:"File:FileModifyDate"` // third
	FileCreateDate int64  `json:"File:FileCreateDate"` // third
	FileName       string `json:"File:FileName"`
	FileType       string `json:"File:FileType"`
	MIMEType       string `json:"File:MIMEType"`

	CreateDate           int64 `json:"EXIF:CreateDate"`           // second
	DateTimeOriginal     int64 `json:"EXIF:DateTimeOriginal"`     // first
	H264DateTimeOriginal int64 `json:"H264:DateTimeOriginal"`     // DateTime for h264
	QTDateTime           int64 `json:"QuickTime:MediaCreateDate"` // DateTime for QuickTime

	GPSLatitude  string `json:"Composite:GPSLatitude"`
	GPSLongitude string `json:"Composite:GPSLongitude"`
}

type Medium struct {
	// meta info by exiftool
	meta *Meta
	//SourceFile string

	//
	//ShootingTime     time.Time
	//ShootingTimeUnix int64
	Adler32 uint32
	SHA256  []byte
	//RelativePath     string
	AbsolutePath string
	Base         string
	os.FileInfo
	imageHash *goimagehash.ImageHash
}

func NewMedium(filename string) *Medium {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil
	}

	stat, err := os.Stat(abs)
	if err != nil || stat == nil || stat.IsDir() || stat.Size() <= 0 {
		return nil
	}

	base := filepath.Base(abs)
	return &Medium{AbsolutePath: abs, Base: base, FileInfo: stat}
}

func (m *Medium) Meta() *Meta {
	if m.meta != nil {
		return m.meta
	}

	args := append(exiftoolFlags, m.AbsolutePath)
	cmd := exec.Command("exiftool", args...)
	zap.L().Debug(cmd.String())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		zap.L().Error(err.Error())
		return nil
	}

	if err := cmd.Start(); err != nil {
		zap.L().Error(err.Error())
		return nil
	}

	decoder := json.NewDecoder(stdout)
	var meta []Meta
	if err := decoder.Decode(&meta); err != nil {
		zap.L().Error(err.Error())
		return nil
	}

	if err := cmd.Wait(); err != nil {
		zap.L().Error(err.Error())
		return nil
	}

	if len(meta) <= 0 {
		zap.L().Info("invalid meta", zap.String("file", m.AbsolutePath))
		return nil
	}

	m.meta = &meta[0]
	return m.meta
}

func (m *Medium) Valid() bool {
	if m.Meta() == nil {
		return false
	}

	return strings.HasPrefix(m.meta.MIMEType, imagePrefix) ||
		strings.HasPrefix(m.meta.MIMEType, videoPrefix)
}

func (m *Medium) ShootingTime() int64 {
	meta := m.Meta()
	if meta == nil {
		return 0
	}

	if meta.DateTimeOriginal > validDataTime {
		return m.meta.DateTimeOriginal
	}

	if meta.H264DateTimeOriginal > validDataTime {
		return meta.H264DateTimeOriginal
	}

	if meta.QTDateTime > validDataTime {
		return meta.QTDateTime
	}

	if meta.CreateDate > validDataTime {
		return meta.CreateDate
	}

	timeFromFilename := extractTime(m.Base)
	if timeFromFilename > validDataTime {
		return timeFromFilename
	}

	if meta.FileModifyDate > meta.FileCreateDate && meta.FileCreateDate > validDataTime {
		return meta.FileCreateDate
	}

	if meta.FileModifyDate > validDataTime {
		return meta.FileModifyDate
	}

	if meta.FileCreateDate > validDataTime {
		return meta.FileCreateDate
	}
	return 0
}

func extractTime(filename string) int64 {
	// try:
	// 	return datetime.strptime(time_str, "%Y:%m:%d %H:%M:%S"), True
	// except Exception:
	// 	pass

	// try:
	// 	return datetime.strptime(time_str, "%Y:%m:%d %H:%M::%S"), True
	// except Exception:
	// 	pass

	// try:
	// 	# 2005-12-02T23:10:04+08:00
	// 	return datetime.strptime(time_str, "%Y-%m-%dT%H:%M:%S+08:00"), False
	// except Exception:
	// 	pass

	// try:
	// 	# 2008-11-18T15:01:14.10+08:00
	// 	return datetime.strptime(time_str, "%Y-%m-%dT%H:%M:%S.%f+08:00"), False
	// except Exception:
	// 	pass

	// try:
	// 	# 2010-10-04T01:52:540Z
	// 	return datetime.strptime(time_str, "%Y-%m-%dT%H:%M:%S0Z"), True
	// except Exception:
	// 	pass

	// return None, False
	return 0
}

func (m *Medium) SumAdler32() {
	if m.Adler32 > 0 {
		return
	}

	file, err := os.Open(m.AbsolutePath)
	if err != nil {
		zap.L().Error("open file", zap.Error(err))
		return
	}
	defer file.Close()

	var data []byte
	if m.FileInfo.Size() > sizeForAdler32 {
		data = make([]byte, sizeForAdler32)
	} else {
		data = make([]byte, m.FileInfo.Size())
	}

	_, err = file.Read(data)
	if err != nil {
		zap.L().Error("read file", zap.Error(err))
		return
	}

	m.Adler32 = adler32.Checksum(data)
}

func (m *Medium) SumSHA256() {
	file, err := os.Open(m.AbsolutePath)
	if err != nil {
		zap.L().Error("open file", zap.Error(err))
		return
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		zap.L().Error("read file", zap.Error(err))
		return
	}

	m.SHA256 = h.Sum(nil)
}

func (m *Medium) PHash() error {
	if m.imageHash != nil {
		return nil
	}

	file, err := os.Open(m.AbsolutePath)
	if err != nil {
		zap.L().Error("open file", zap.Error(err))
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		zap.L().Debug("image.Decode", zap.String("file", m.AbsolutePath), zap.Error(err))
		return err
	}

	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		zap.L().Debug("goimagehash.PerceptionHash", zap.String("file", m.AbsolutePath), zap.Error(err))
		return err
	}

	m.imageHash = hash
	return nil
}

func (m *Medium) Same(other *Medium) bool {

	if m.FileInfo.Size() != other.FileInfo.Size() {
		return false
	}

	m.SumAdler32()
	other.SumAdler32()

	if m.Adler32 != other.Adler32 {
		return m.sameImage(other)
	}

	m.SumSHA256()
	other.SumSHA256()
	if len(m.SHA256) != len(other.SHA256) {
		return m.sameImage(other)
	}

	for i := 0; i < len(m.SHA256); i++ {
		if m.SHA256[i] != other.SHA256[i] {
			return m.sameImage(other)
		}
	}
	return true
}

func (m *Medium) sameImage(other *Medium) bool {
	if m.ShootingTime() > 0 && other.ShootingTime() > 0 && m.ShootingTime() == other.ShootingTime() {
		return true
	}
	//m.MagickWand()
	//other.MagickWand()
	//
	//if m.mw != nil && other.mw != nil {
	//	_, distortion := m.mw.CompareImages(other.mw, imagick.METRIC_PERCEPTUAL_HASH_ERROR)
	//	logger.FileInfo("different image", zap.String("file1", m.AbsolutePath), zap.String("file2", other.AbsolutePath), zap.Float64("distortion", distortion))
	//}

	if m.PHash() == nil && other.PHash() == nil {
		if dis, err := m.imageHash.Distance(other.imageHash); err == nil {
			//if dis != 0 {
			//	logger.Info("different image", zap.String("file1", m.AbsolutePath), zap.String("file2", other.AbsolutePath), zap.Int("distance", dis))
			//}
			return dis == 0
		}
	}
	return false
}
