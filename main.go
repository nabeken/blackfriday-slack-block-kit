package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/nabeken/blackfriday-slack-block-kit/blockkit"
	bf "github.com/russross/blackfriday/v2"
)

func main() {
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	parser := bf.New(bf.WithExtensions(bf.CommonExtensions))
	conv := blockkit.NewConverter(parser.Parse([]byte(input)))

	json.NewEncoder(os.Stdout).Encode(conv.Convert())
}
