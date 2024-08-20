package subcmd

import (
	"flag"
	"fmt"
	"os"
)

func New(name, doc string) *Subcommand {
	sc := &Subcommand{
		FlagSet: flag.NewFlagSet(name, flag.ContinueOnError),
	}
	sc.FlagSet.Usage = func() {
		argSuffix := ""
		if sc.arg != nil {
			argSuffix = fmt.Sprintf(" <%s>", sc.arg.name)
		}
		fmt.Fprintf(os.Stderr, "\n"+doc+"\n\n")
		fmt.Fprintf(os.Stderr, "  genres %s [flags]%s\n\n", name, argSuffix)
		fmt.Fprintf(os.Stderr, "flags:\n")
		sc.FlagSet.PrintDefaults()
		if sc.arg != nil {
			fmt.Fprintf(os.Stderr, "  <%s> %s\n", sc.arg.name, sc.arg.typename)
			fmt.Fprintf(os.Stderr, "  \t%s\n", sc.arg.usage)
		}
	}
	return sc
}

type Subcommand struct {
	*flag.FlagSet
	arg *arg
}

type arg struct {
	name     string
	typename string
	usage    string
}

func (sc *Subcommand) SetArg(name, typname, usage string) *Subcommand {
	sc.arg = &arg{name, typname, usage}
	return sc
}
