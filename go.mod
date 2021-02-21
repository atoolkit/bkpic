module github.com/enjoypi/bkpic

go 1.15

replace github.com/enjoypi/gobsdiff => ../enjoypi/gobsdiff

require (
	github.com/corona10/goimagehash v1.0.3
	github.com/enjoypi/gobsdiff v0.0.0-00010101000000-000000000000
	github.com/enjoypi/gojob v0.0.0-20210120062315-66a1361e0c87
	github.com/pkg/errors v0.8.1
	github.com/restic/chunker v0.4.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	go.uber.org/zap v1.16.0
	gopkg.in/yaml.v2 v2.4.0
)
