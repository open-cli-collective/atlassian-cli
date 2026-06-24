package page

import (
	"fmt"
	"os"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

var stdinIsTTY = func() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func hasInjectedStdin(opts *root.Options) bool {
	return opts != nil && opts.Stdin != nil && opts.Stdin != os.Stdin
}

func hasPipedOSStdin(opts *root.Options) bool {
	return opts != nil && opts.Stdin == os.Stdin && !stdinIsTTY()
}

func hasContentSource(opts *root.Options, file string, editor bool) bool {
	return file != "" || editor || hasInjectedStdin(opts) || hasPipedOSStdin(opts)
}

func errMissingContentSource() error {
	return fmt.Errorf("page content source is required: use --file, --editor, or pipe content via stdin")
}
