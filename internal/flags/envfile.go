package flags

import "github.com/spf13/pflag"

func AddEnvFileFlag(fs *pflag.FlagSet, target *string) {
	fs.StringVarP(target, "env-file", "o", "", "write/merge exports into this .env file")
}
