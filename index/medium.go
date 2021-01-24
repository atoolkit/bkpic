package index

import (
	"crypto/sha256"
	"hash/adler32"
	"image"
	"io"
	"os"
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
	validDataTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
)

type Medium struct {
	// meta info by exiftool
	SourceFile     string
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

	//
	ShootingTime     time.Time
	ShootingTimeUnix int64
	Adler32          uint32
	Sha              string
	Size             int64
	RelativePath     string
	AbsolutePath     string
	os.FileInfo
	sha256    []byte
	imageHash *goimagehash.ImageHash
}

func (m *Medium) init(basepath string) {
	if m.Valid() {
		var err error
		m.RelativePath, err = filepath.Rel(basepath, m.Directory)
		if err != nil {
			zap.L().Error(err.Error())
		}
		m.RelativePath = filepath.ToSlash(m.RelativePath)
		m.SourceFile = filepath.ToSlash(m.SourceFile)
		m.ShootingTimeUnix = m.shootingTime()
		m.ShootingTime = time.Unix(m.ShootingTimeUnix, 0)
	}
}

func (m *Medium) Valid() bool {
	return strings.HasPrefix(m.MIMEType, imagePrefix) ||
		strings.HasPrefix(m.MIMEType, videoPrefix)
}

func (m *Medium) shootingTime() int64 {
	if m.DateTimeOriginal > validDataTime {
		return m.DateTimeOriginal
	}

	if m.H264DateTimeOriginal > validDataTime {
		return m.H264DateTimeOriginal
	}

	if m.QTDateTime > validDataTime {
		return m.QTDateTime
	}

	if m.CreateDate > validDataTime {
		return m.CreateDate
	}

	timeFromFilename := extractTime(m.SourceFile)
	if timeFromFilename > validDataTime {
		return timeFromFilename
	}

	if m.FileModifyDate > m.FileCreateDate && m.FileCreateDate > validDataTime {
		return m.FileCreateDate
	}

	if m.FileModifyDate > validDataTime {
		return m.FileModifyDate
	}

	if m.FileCreateDate > validDataTime {
		return m.FileCreateDate
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

	m.sha256 = h.Sum(nil)
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
		return m.SameImage(other)
	}

	m.SumSHA256()
	other.SumSHA256()
	if len(m.sha256) != len(other.sha256) {
		return m.SameImage(other)
	}

	for i := 0; i < len(m.sha256); i++ {
		if m.sha256[i] != other.sha256[i] {
			return m.SameImage(other)
		}
	}
	return true
}

func (m *Medium) SameImage(other *Medium) bool {
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
