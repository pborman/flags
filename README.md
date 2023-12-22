# options [![GoDoc](https://godoc.org/github.com/pborman/flags?status.svg)](http://godoc.org/github.com/pborman/flags)

Flags provides an elegant structured mechanism for declaring flags
in Go programs.  It is a simplified version of the
github.com/pborman/options package.  By default it wraps the standard
flag package but can easily be adjusted to wrap any similar flag
package.

Below is a simple program that uses this package:

```
package main

import (
	"fmt"
	"time"

	"github.com/pborman/flags"
)

type options struct {
	Name    string        `flag:"--name=NAME      name of the widget"`
	Count   int           `flag:"--count=COUNT    number of widgets"`
	Verbose bool          `flag:"-v               be verbose"`
	N       int           `flag:"-n=NUMBER        set n to NUMBER"`
	Timeout time.Duration `flag:"--timeout        duration of run"`
	Lazy    string
}
var opts = options {
	Name: "gopher",
}

func main() {
	args := flags.RegisterAndParse(&opts)

	if opts.Verbose {
		fmt.Printf("Command line parameters: %q\n", args)
	}
	fmt.Printf("Name: %s\n", opts.Name)
}
```

The fields in the structure must be compatible with one of the
following types:

*  bool
*  int
*  int64
*  float64
*  string
*  uint
*  uint64
*  []string
*  Value (*interface { String() string; Set(string) error }*)
*  time.Duration

The following type is compatible with a string:

```
type Name string
```

The following are various ways to use the above declaration.

```
// Register opts, parse the command line, and set args to the
// remaining command line parameters
args := flags.RegisterAndParse(&opts)

// Validate opts.
err := flag.Validate(&opts)
if err != nil { ... }

// Register opts as command line options.
flags.Register(&opts)

// Register options to a new flag set.
set := flags.NewFlagSet("")
flags.RegisterSet(&options, set)

// Register a new instance of opts
vopts, set := flags.RegisterNew(&opts)
newOpts := vopts.(*options)
```
