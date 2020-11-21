package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/nabeken/blackfriday-slack-block-kit/blockkit"
	bf "github.com/russross/blackfriday/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func versionInfo() {
	fmt.Fprintf(os.Stderr, "Version: %s\nCommit: %s\nBuiltAt: %s\n", version, commit, date)
}

func main() {
	debug := flag.Bool("debug", false, "Enable debug output")
	showVersion := flag.Bool("version", false, "show version info")

	origUsage := flag.Usage
	flag.Usage = func() {
		origUsage()
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "== Build Info ==\n")
		versionInfo()
	}

	flag.Parse()

	if *showVersion {
		flag.Usage()
		os.Exit(0)
	}

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	parser := bf.New(bf.WithExtensions(bf.CommonExtensions))
	conv := blockkit.NewConverter(parser.Parse([]byte(input)))
	if *debug {
		conv.Debug()
	}

	json.NewEncoder(os.Stdout).Encode(conv.Convert())
}
