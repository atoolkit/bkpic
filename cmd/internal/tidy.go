package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/enjoypi/bkpic/index"
	"github.com/enjoypi/bkpic/windows"
	"github.com/sirupsen/logrus"
)

type TidyConfig struct {
	Debug  bool
	DryRun bool `mapstructure:"dry-run"`
	Move   bool
	Output string
}

var (
	usefulPattern, _ = regexp.Compile(`[^\w\s\.\:\/\-]+`)
	uselessDirs      = map[string]bool{
		"照片":             true,
		"手机照相":           true,
		"个人":             true,
		"视频":             true,
		"贝贝":             true,
		"妈妈爸爸":           true,
		"贝贝手机视频":         true,
		"手机9月5日导出":       true,
		"扫描照片":           true,
		"照片1":            true,
		"妈妈手机照片":         true,
		"来自：huawei a199": true,
		"爷爷手机导出":         true,
		"爸爸手机照片":         true,
		"7-8月":           true,
		"2013春节":         true,
		"照片2":            true,
		"新建文件夹":          true,
		"2013国庆":         true,
		"新建公文包":          true,
		"2014春节":         true,
		"14年":            true,
		"ww手机照片":         true,
		"本机照片":           true,
		"8月":             true,
		"9月":             true,
		"家庭照片":           true,
		"姜微":             true,
		"成都照片":           true,
		"姜蔚":             true,
		"贝贝照片":           true,
		"老照片":            true,
		"2009春节":         true,
		"2009年5月至7月照片":   true,
		"姜":              true,
		"2011.8贝贝":       true,
		"小年2013-05":      true,
		"2015年09月":       true,
		"2015年08月":       true,
		"2016年02月":       true,
		"蔚家":             true,
		"咱家":             true,
		"08年":            true,
		"09年":            true,
		"10年":            true,
		"11年":            true,
	}
)

func Tidy(c *TidyConfig, inputs []string) error {
	absOutput, err := checkOutput(filepath.Clean(c.Output))
	if err != nil {
		return err
	}
	c.Output = absOutput

	for _, in := range inputs {
		input := filepath.Clean(in)
		if err := doTidy(c, input, absOutput); err != nil {
			logrus.Warn(err, " in ", input)
		}
	}
	return nil
}

func checkOutput(output string) (string, error) {
	abs, err := filepath.Abs(output)
	if err != nil {
		return "", err
	}
	abs = filepath.ToSlash(abs)

	if err := os.MkdirAll(abs, os.FileMode(0700)); err != nil {
		return "", err
	}
	return abs, nil
}

func checkInput(src string) (string, error) {
	abs, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	abs = filepath.ToSlash(abs)

	fileInfo, err := os.Stat(abs)
	if err != nil {
		return "", err
	}

	if !fileInfo.IsDir() {
		return "", fmt.Errorf("%s is not directory", src)
	}

	return abs, nil
}

func doTidy(c *TidyConfig, in string, out string) error {
	absIn, err := checkInput(in)
	if err != nil {
		return err
	}

	if absIn == out {
		return fmt.Errorf("input is same with output %s", out)
	}

	media, err := index.NewIndex(absIn)
	if err != nil {
		return err
	}
	if c.Debug {
		media.Save()
	}

	files := media.Files()
	for _, src := range files {
		m := media.Media[src]

		out, outDir := getOutPath(m, out)
		if out == src {
			continue
		}

		logrus.Info(src, "\t=>\t", out)
		if c.DryRun {
			continue
		}

		var err error
		for i := 1; i <= 10; i++ {
			err = do(c, src, outDir, out)
			if err == os.ErrExist {
				ext := filepath.Ext(out)
				out = strings.TrimSuffix(out, ext) + fmt.Sprintf("_%d", i) + ext
				continue
			}
			if err != nil {
				break
			}
		}

		if err != nil {
			logrus.Warn(err, " in ", src)
		}
	}
	return nil
}

func getOutPath(src *index.Medium, absTgt string) (string, string) {
	tgtDir := fmt.Sprintf("%04d/%02d",
		src.ShootingTime.Year(),
		src.ShootingTime.Month())

	// match han
	dirs := strings.Split(src.RelativePath, "/")
	n := len(dirs)
	if n > 0 {
		for i := n - 1; i >= 0; i-- {
			dir := dirs[i]
			if ok := uselessDirs[strings.ToLower(dir)]; ok {
				continue
			}

			if usefulPattern.MatchString(dir) {
				tgtDir = tgtDir + "/" + dir
				break
			}
		}
	}
	tgtPath := tgtDir + "/" + src.FileName
	return absTgt + "/" + tgtPath, absTgt + "/" + tgtDir
}

func sum256(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func do(c *TidyConfig, src, outDir, out string) error {
	outFileInfo, err := os.Stat(out)
	// 目标文件不存在，直接移动
	if os.IsNotExist(err) {

		if err := os.MkdirAll(outDir, os.FileMode(0700)); err != nil {
			return err
		}

		if c.Move {
			return windows.Move(src, out)
		}

		return windows.Copy(src, out)
	}

	if err != nil {
		return err
	}

	srcFileInfo, err := os.Stat(src)
	if err != nil {
		return nil
	}

	// 目标文件存在则判断是否一致
	if srcFileInfo.Size() != outFileInfo.Size() {
		return os.ErrExist
	}

	srcHash, err := sum256(src)
	if err != nil {
		return err
	}

	outHash, err := sum256(out)
	if err != nil {
		return err
	}

	if srcHash != outHash {
		return os.ErrExist
	}

	if c.Move {
		_ = os.Remove(src)
	}
	return nil
}
