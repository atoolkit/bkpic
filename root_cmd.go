package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/enjoypi/bkpic/index"
	"github.com/enjoypi/bkpic/windows"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

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

func rootAction(c *cli.Context) error {
	if c.Bool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	narg := c.NArg()
	if narg < 1 {
		return cli.ShowAppHelp(c)
	}

	args := c.Args()
	if c.NArg() == 1 {
		return cli.ShowAppHelp(c)
	}

	return move(c, args[0:narg-1], args[narg-1])
}

func move(c *cli.Context, srcs []string, tgtDir string) error {
	absTgtDir, err := filepath.Abs(tgtDir)
	if err != nil {
		return err
	}
	absTgtDir = filepath.ToSlash(absTgtDir)

	tgtMode := os.FileMode(0700)
	if err := os.MkdirAll(absTgtDir, tgtMode); err != nil {
		return err
	}

	startTime := time.Now()

	doneCount := 0
	errorCount := 0
	years := make(map[int]bool)

	for _, src := range srcs {
		absSrcDir, err := filepath.Abs(src)
		if err != nil {
			log.Warnln(err)
			//多个src文件夹时其中一个没有继续执行其他文件夹
			continue
		}
		absSrcDir = filepath.ToSlash(absSrcDir)

		if absSrcDir == absTgtDir {
			log.Warn("source can not be same with target")
			continue
		}

		srcStat, err := os.Stat(absSrcDir)
		if err != nil {
			log.Warnln(err)
			continue
		}

		if !srcStat.IsDir() {
			log.Warnf("%s is not directory!", src)
			continue
		}

		log.Infof("%s 处理中......", absSrcDir)

		srcIndex, err := index.NewIndex(absSrcDir)
		if err != nil {
			log.Warnln(err)
			continue
		}

		if len(srcIndex.Media) <= 0 {
			continue
		}
		defer srcIndex.Save()

		for _, relSrcPath := range srcIndex.Files() {
			srcMedium := srcIndex.Media[relSrcPath]

			absSrcPath := absSrcDir + "/" + relSrcPath
			absTgtPath, dir := getTargetPath(srcMedium, absTgtDir)

			if absSrcPath == absTgtPath {
				//  不需要移动
				continue
			}

			// 源文件是否正常
			srcFileStat, err := os.Stat(absSrcPath)
			if err != nil {
				log.Warnln(err)
				continue
			}

			if !c.Bool("dry-run") {
				if err := os.MkdirAll(dir, tgtMode); err != nil {
					log.Warn(err)
					continue
				}
			}

			var srcSha256 string
			nameChanged := false
			for i := 1; i <= 10; i++ {
				if err, srcSha256 = do(c, srcFileStat, srcSha256, absSrcPath, absTgtPath); err == nil {
					// 有同名文件则输出日志
					if nameChanged {
						log.Info(absSrcPath, "\t改名为\t", absTgtPath)
					} else {
						log.Debug(absSrcPath, "\t=>\t", absTgtPath)
					}
					// 移动成功，结束
					years[srcMedium.ShootingTime.Year()] = true
					doneCount++
					break
				}
				nameChanged = true
				ext := filepath.Ext(absTgtPath)
				absTgtPath = strings.TrimSuffix(absTgtPath, ext) + fmt.Sprintf("_%d", i) + ext
			}

			if err != nil {
				log.Warn(err)
			}
		}
	}

	fmt.Printf("\n%d个文件完成在%f内, years %v.\n%d 错误!\n\n", doneCount, time.Now().Sub(startTime).Seconds(), years, errorCount)
	return nil
}

func getTargetPath(src *index.Medium, absTgt string) (string, string) {
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

func sum256(filename string) (error, string) {
	f, err := os.Open(filename)
	if err != nil {
		return err, ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err, ""
	}

	return nil, hex.EncodeToString(h.Sum(nil))
}

func do(c *cli.Context, srcFileStat os.FileInfo, srcSha256, absSrcPath, absTgtPath string) (error, string) {

	tgtFileStat, err := os.Stat(absTgtPath)
	// 目标文件不存在，直接移动
	if os.IsNotExist(err) {
		if c.Bool("dry-run") {
			return nil, srcSha256
		}
		if c.Bool("move") {
			err = windows.Move(absSrcPath, absTgtPath)
		} else {
			err = windows.Copy(absSrcPath, absTgtPath)
		}
		return err, srcSha256
	}

	// 目标文件存在则判断是否一致
	if err == nil {
		if srcFileStat.Size() == tgtFileStat.Size() {
			srcHash := srcSha256
			if srcHash == "" {
				if err, srcHash = sum256(absSrcPath); err != nil {
					return err, srcSha256
				}
			}

			var tgtHash string
			if err, tgtHash = sum256(absTgtPath); err != nil {
				return err, srcHash
			}

			if srcHash == tgtHash {
				log.Infof("%s\t已经存在", absSrcPath)
				return nil, srcHash
			}

		}

		return fmt.Errorf("%s is exists!", absTgtPath), srcSha256
	}

	return err, srcSha256
}
