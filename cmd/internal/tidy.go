package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/enjoypi/bkpic/fs"
	"github.com/enjoypi/bkpic/index"
)

type TidyConfig struct {
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
	absOutput, err := filepath.Abs(c.Output)
	if err != nil {
		return err
	}

	if err := checkOutput(absOutput); err != nil {
		return err
	}
	c.Output = absOutput

	idx, err := index.NewIndex(absOutput)
	if err != nil {
		return err
	}

	for _, in := range inputs {
		input, err := filepath.Abs(in)
		if err != nil {
			zap.L().Error("invalid file path", zap.Error(err))
			continue
		}

		if err := doTidy(c, input, idx); err != nil {
			zap.S().Warn(err, " in ", input)
		}
	}
	return nil
}

func checkOutput(output string) error {

	stat, err := os.Stat(output)
	if stat == nil {
		if err := os.MkdirAll(output, os.FileMode(0700)); err != nil {
			return err
		}
		stat, err = os.Stat(output)
	}

	if err != nil {
		return err
	}

	if !stat.IsDir() {
		return index.ErrNotDirectory
	}

	return nil
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

func doTidy(c *TidyConfig, in string, idx *index.Index) error {
	absIn, err := checkInput(in)
	if err != nil {
		return err
	}

	absOut := idx.AbsoluteDirectory()
	if absIn == absOut {
		return fmt.Errorf("input is same with output %s", absIn)
	}

	var count int
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		src := index.NewMedium(path)
		if src == nil {
			zap.L().Info("invalid medium", zap.String("file", path))
			return nil
		}

		same := idx.Same(src)
		if same != nil {
			zap.L().Info("file already exists", zap.String("source", path), zap.String("same", same.AbsolutePath))
			return nil
		}

		if !src.Valid() {
			return index.ErrInvalidMedium
		}

		out, outDir := getOutPath(src, absOut)
		if out == path {
			return nil
		}

		zap.L().Info(fmt.Sprintf("%s\t=>\t%s", path, out))
		for i := 1; i <= 10; i++ {
			err = do(c, path, outDir, out)
			if err == os.ErrExist {
				ext := filepath.Ext(out)
				out = strings.TrimSuffix(out, ext) + fmt.Sprintf("_%d", i) + ext
				continue
			} else {
				break
			}
		}

		if err != nil {
			zap.S().Warn(err, " in ", path)
		} else {
			count++
		}

		return nil
	}

	if err := filepath.Walk(absIn, walk); err != nil {
		return err
	}

	zap.S().Infof("已完成。总文件：%d，成功：%d", len(files), count)
	return nil
}

func getOutPath(src *index.Medium, absTgt string) (string, string) {
	if src.ShootingTime() <= 0 {
		return "", ""
	}

	shooting := time.Unix(src.ShootingTime(), 0)
	tgtDir := fmt.Sprintf("%04d/%02d", shooting.Year(), shooting.Month())

	// match han
	//dirs := strings.Split(src.RelativePath, "/")
	//n := len(dirs)
	//if n > 0 {
	//	for i := n - 1; i >= 0; i-- {
	//		dir := dirs[i]
	//		if ok := uselessDirs[strings.ToLower(dir)]; ok {
	//			continue
	//		}
	//
	//		if usefulPattern.MatchString(dir) {
	//			tgtDir = tgtDir + "/" + dir
	//			break
	//		}
	//	}
	//}
	tgtPath := tgtDir + "/" + src.Base
	return absTgt + "/" + tgtPath, absTgt + "/" + tgtDir
}

func do(c *TidyConfig, src, outDir, out string) error {
	_, err := os.Stat(out)
	// 目标文件不存在，直接移动
	if os.IsNotExist(err) {
		if c.DryRun {
			return nil
		}

		if err := os.MkdirAll(outDir, os.FileMode(0700)); err != nil {
			return err
		}

		if c.Move {

			return fs.Move(src, out)
		}

		return fs.Copy(src, out)
	}

	if err != nil {
		return err
	}

	if c.Move && !c.DryRun {
		_ = os.Remove(src)
	}
	return nil
}
