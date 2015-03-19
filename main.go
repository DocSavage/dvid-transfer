// Transfer data between DVID instances

package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	// Display usage if true.
	showHelp = flag.Bool("help", false, "")

	// Run in verbose mode if true.
	runVerbose = flag.Bool("verbose", false, "")
)

const helpMessage = `
dvid-transfer moves data from one DVID server to another using HTTP API calls.

Usage: dvid-transfer [options] src dst

  where src = URL for the source data in form http://host/api/node/<uuid>/<dataname>
    and dst = URL for the destination.

  If the destination UUID doesn't already exist, there is an error.
  If the destination data name doesn't exist, it is created.

	  -verbose    (flag)    Run in verbose mode.
      -h, -help   (flag)    Show help message

`

func main() {
	flag.BoolVar(showHelp, "h", false, "Show help message")
	flag.Usage = func() {
		fmt.Printf(helpMessage)
	}
	flag.Parse()

	if *showHelp || flag.NArg() != 2 {
		flag.Usage()
		os.Exit(0)
	}

	src := flag.Args()[0]
	dst := flag.Args()[1]

	transferData(src, dst)
}
