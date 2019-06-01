package index

import (
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	imagePrefix = "image/"
	videoPrefix = "video/"
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
}

func (m *Medium) init(basepath string) {
	if m.Valid() {
		var err error
		m.RelativePath, err = filepath.Rel(basepath, m.Directory)
		if err != nil {
			log.Warn(err)
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
