package index

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"hash/adler32"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/enjoypi/gordiff/wrapper"
	"github.com/icedream/go-bsdiff"
	"go.uber.org/zap"
)

const (
	sizeForAdler32 = 4096
	audioPrefix    = "audio/"
	imagePrefix    = "image/"
	videoPrefix    = "video/"
)

var (
	exiftoolFlags = []string{"-a", "-charset", "FileName=UTF8", "-d", "%s", "-ee", "--ext", "json", "-G", "-j", "-L", "-q", "-r", "-sort"}
	validDataTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
)

type Meta struct {
	SourceFile string `json:SourceFile`
	//Directory      string `json:"File:Directory"`
	FileModifyDate int64 `json:"File:FileModifyDate"` // third
	FileCreateDate int64 `json:"File:FileCreateDate"` // third
	//FileName       string `json:"File:FileName"`
	FileType    string `json:"File:FileType"`
	ImageHeight int64  `json:"File:ImageHeight"`
	ImageWidth  int64  `json:"File:ImageWidth"`
	MIMEType    string `json:"File:MIMEType"`

	EXIFCreateDate       int64  `json:"EXIF:CreateDate"` // second
	EXIFModifyDate       int64  `json:"EXIF:ModifyDate"`
	DateTimeOriginal     int64  `json:"EXIF:DateTimeOriginal"`     // first
	Model                string `json:"EXIF:Model"`                //  camera model
	H264DateTimeOriginal int64  `json:"H264:DateTimeOriginal"`     // DateTime for h264
	QTDateTime           int64  `json:"QuickTime:MediaCreateDate"` // DateTime for QuickTime
	XMPPhotoId           string `json:"XMP:PhotoId"`

	GPSLatitude  string `json:"Composite:GPSLatitude"`
	GPSLongitude string `json:"Composite:GPSLongitude"`
}

type Medium struct {
	// meta info by exiftool
	meta     *Meta
	metaDone bool

	//
	//ShootingTime     time.Time
	//ShootingTimeUnix int64
	Adler32  uint32
	SHA256   []byte
	FullPath string
	os.FileInfo
	imageHash *goimagehash.ImageHash
	signFile  string
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

	return &Medium{FullPath: abs, FileInfo: stat}
}

func (m *Medium) Meta() *Meta {
	if m.metaDone {
		return m.meta
	}
	// just do once
	m.metaDone = true

	args := append(exiftoolFlags, m.FullPath)
	cmd := exec.Command("exiftool", args...)

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
	var meta []*Meta
	if err := decoder.Decode(&meta); err != nil {
		zap.L().Error(err.Error())
		return nil
	}

	if err := cmd.Wait(); err != nil {
		zap.L().Error(err.Error())
		return nil
	}

	if len(meta) <= 0 {
		zap.L().Info("invalid meta", zap.String("file", m.FullPath))
		return nil
	}

	m.meta = meta[0]
	return m.meta
}

func (m *Medium) Valid() bool {
	if m.Meta() == nil {
		return false
	}

	return strings.HasPrefix(m.meta.MIMEType, audioPrefix) ||
		strings.HasPrefix(m.meta.MIMEType, imagePrefix) ||
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

	if meta.EXIFCreateDate > validDataTime {
		return meta.EXIFCreateDate
	}

	timeFromFilename := extractTime(m.FileInfo.Name())
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

	file, err := os.Open(m.FullPath)
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
	file, err := os.Open(m.FullPath)
	if err != nil {
		zap.L().Error("open file", zap.Error(err))
		return
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		zap.L().Info("read file", zap.Error(err))
		return
	}

	m.SHA256 = h.Sum(nil)
}

func (m *Medium) PHash() error {
	if m.imageHash != nil {
		return nil
	}

	file, err := os.Open(m.FullPath)
	if err != nil {
		zap.L().Info("open file", zap.Error(err))
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		zap.L().Info("image.Decode",
			zap.String("mime", m.meta.MIMEType),
			zap.String("file", m.FullPath), zap.Error(err))
		return err
	}

	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		zap.L().Info("goimagehash.PerceptionHash", zap.String("file", m.FullPath), zap.Error(err))
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
		return m.same(other)
	}

	m.SumSHA256()
	other.SumSHA256()
	if len(m.SHA256) != len(other.SHA256) {
		return m.same(other)
	}

	for i := 0; i < len(m.SHA256); i++ {
		if m.SHA256[i] != other.SHA256[i] {
			return m.same(other)
		}
	}
	return true
}

func (m *Medium) same(other *Medium) bool {
	if !m.Valid() {
		return false
	}

	if strings.HasPrefix(m.meta.MIMEType, imagePrefix) {
		return m.sameImage(other)
	}

	return m.sameChunk(other)
}

func (m *Medium) sameImage(other *Medium) bool {

	if m.sameMeta(other) {
		return true
	}

	if m.meta == nil || other.meta == nil {
		return false
	}

	if !strings.HasPrefix(m.meta.MIMEType, imagePrefix) {
		return false
	}

	if !strings.HasPrefix(other.meta.MIMEType, imagePrefix) {
		return false
	}

	if m.PHash() == nil && other.PHash() == nil {
		if dis, err := m.imageHash.Distance(other.imageHash); err == nil {
			//if dis != 0 {
			//	logger.Info("different image", zap.String("file1", m.FullPath), zap.String("file2", other.FullPath), zap.Int("distance", dis))
			//}
			return dis == 0
		}
	}
	return false
}

func (m *Medium) sameMeta(other *Medium) bool {
	meta := m.Meta()
	otherMeta := other.Meta()
	if meta == nil || otherMeta == nil {
		return false
	}

	if meta.Model != "" && meta.Model == otherMeta.Model &&
		meta.ImageHeight > 0 && meta.ImageHeight == otherMeta.ImageHeight &&
		meta.ImageWidth > 0 && meta.ImageWidth == otherMeta.ImageWidth &&
		m.ShootingTime() > 0 && m.ShootingTime() == other.ShootingTime() {
		return true
	}
	return false
}

func (m *Medium) sameChunk(other *Medium) bool {
	if m.signFile == "" {
		m.signFile, _ = wrapper.RSSig(m.FullPath)
	}

	if m.signFile != "" {
		if deltaPath, err := wrapper.RSDelta(m.FullPath, m.signFile, other.FullPath); err == nil {
			if stat, err := os.Stat(deltaPath); err == nil {
				ratio := float64(stat.Size()) / float64(m.FileInfo.Size())
				if ratio <= 0.1001 {
					return true
				} else if ratio > 0.5 {
					return false
				}
				// others to bsdiff
			}
		}
	}

	file, err := os.Open(m.FullPath)
	if err != nil {
		zap.L().Info("open file", zap.Error(err))
		return false
	}
	defer file.Close()

	ofile, err := os.Open(other.FullPath)
	if err != nil {
		zap.L().Info("open file", zap.Error(err))
		return false
	}
	defer ofile.Close()

	patch := new(bytes.Buffer)

	zap.L().Debug("start bsdiff")
	if err := bsdiff.Diff(file, ofile, patch); err != nil {
		zap.L().Debug("bsdiff error", zap.Error(err))
		return false
	}

	//var same float64
	//for {
	//	lb, lerr := lhs.ReadByte()
	//	if lerr != nil {
	//		break
	//	}
	//
	//	rb, rerr := rhs.ReadByte()
	//	if rerr != nil {
	//		break
	//	}
	//
	//	if lb == rb {
	//		same++
	//	}
	//}
	//m.sumChunk()
	//other.sumChunk()
	//
	//if len(m.chunks) != len(other.chunks) {
	//	return false
	//}
	//
	//for i, chunk := range m.chunks {
	//	if chunk != other.chunks[i] {
	//		return false
	//	}
	//}
	diff := float64(patch.Len()) / (float64(m.FileInfo.Size()))
	zap.L().Debug("sameChunk",
		zap.String("lhs", m.FullPath),
		zap.String("rhs", other.FullPath),
		zap.Int64("size", m.FileInfo.Size()),
		//zap.Float64("same", same/(float64(m.FileInfo.Size()))),
		zap.Int("patch size", patch.Len()),
		zap.Float64("diff", diff),
	)
	//return same/float64(m.FileInfo.Size()) > 0.8
	return diff <= 0.2001
}
