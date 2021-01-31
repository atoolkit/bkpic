package tidy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/enjoypi/bkpic/index"
	"github.com/enjoypi/gojob"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type config struct {
	Path2rm map[string]bool
}

func Run(v *viper.Viper, args []string) error {
	var cfg config

	f, err := os.Open("rmdup.yaml")
	if err == nil {
		d := yaml.NewDecoder(f)
		if err := d.Decode(&cfg); err != nil {
			zap.L().Error(err.Error())
		}
	} else {
		zap.L().Error(err.Error())
	}

	idx := index.NewEmptyIndex()
	for _, arg := range args {
		if err := idx.Walk(arg); err != nil {
			return err
		}
	}

	files := idx.GetMediaBySize()
	keys := make([]int, 0)
	for k, hashes := range files {
		if len(hashes) > 1 {
			keys = append(keys, int(k))
		}
	}
	sort.Ints(keys)
	//reverse(keys)
	//sort.Sort(sort.Reverse(sort.IntSlice(keys)))

	m := gojob.NewManager(int64(runtime.GOMAXPROCS(0)))
	for i := 0; i < len(keys); i++ {
		values := context.WithValue(m.Context, "size", int64(keys[i]))
		m.Go(func(ctx context.Context, id gojob.TaskID) error {
			size := ctx.Value("size").(int64)
			zap.L().Debug("started", zap.Int32("taskID", id), zap.Int64("size", size))

			hashes := files[size]

			first := hashes[0]
			same := make([]string, 0)
			for j := 1; j < len(hashes); j++ {
				other := hashes[j]
				if first.Same(hashes[j]) {
					// 加入第一个
					if len(same) <= 0 {
						same = append(same, first.FullPath)
					}

					same = append(same, other.FullPath)
				}
			}

			if len(same) > 0 {
				logRM(same, &cfg)
			} else {
				logDupFiles(hashes)
			}
			return nil
		}, values, nil)
	}

	m.Wait()

	return nil
}

func logDupFiles(hashes []*index.Medium) {
	buf := bytes.NewBufferString("")
	for _, v := range hashes {
		buf.WriteString("#rm \"")
		buf.WriteString(v.FullPath)
		buf.WriteString("\"\n")
	}
	fmt.Println(buf.String())
}

func match(fullpath string, path2match map[string]bool) bool {

	base := path.Base(fullpath)
	ext := path.Ext(fullpath)
	noExt := strings.TrimSuffix(base, ext)
	if strings.HasSuffix(noExt, "(2)") || strings.HasSuffix(noExt, "_1") || strings.HasSuffix(noExt, "_1_2") {
		return true
	}

	dirs := strings.Split(fullpath, "/")
	for _, dir := range dirs[:len(dirs)-1] {
		if len(dir) <= 0 {
			continue
		}
		if _, ok := path2match[dir]; ok {
			return true
		}
	}

	return false
}

// StringSlice attaches the methods of Interface to []string, sorting in increasing order.
type customSlice []string

func (p customSlice) Len() int { return len(p) }
func (p customSlice) Less(i, j int) bool {
	// compare length first
	if utf8.RuneCountInString(p[i]) < utf8.RuneCountInString(p[j]) {
		return true
	}
	return p[i] < p[j]
}
func (p customSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func logRM(files []string, cfg *config) {
	sort.Sort(customSlice(files))
	// rm shortest path
	rm := 0
	for i := 0; i < len(files); i++ {
		if match(files[i], cfg.Path2rm) {
			rm = i
			break
		}
	}
	buf := bytes.NewBufferString("")
	for i := 0; i < len(files); i++ {
		if i != rm {
			buf.WriteString("#")
		}
		buf.WriteString("rm \"")
		buf.WriteString(files[i])
		buf.WriteString("\"\n")
	}

	fmt.Println(buf.String())
}

func walk(path string, info os.FileInfo, err error) error {
	return nil
}
