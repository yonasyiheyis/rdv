package flags

import "github.com/spf13/pflag"

// --no-prompt: disable interactive prompts; read values from flags
func AddNoPromptFlag(fs *pflag.FlagSet, target *bool) {
	fs.BoolVar(target, "no-prompt", false, "disable interactive prompts (use flags instead)")
}
