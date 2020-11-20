package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/k0kubun/pp"
	bf "github.com/russross/blackfriday/v2"
)

const (
	textTypeMrkdwn = "mrkdwn"
	textTypePlain  = "plain_text"

	tagItalic     = "_"
	tagStrong     = "*"
	tagStrike     = "~"
	tagItem       = "-"
	tagLink       = "<"
	tagLinkClose  = ">"
	tagCode       = "`"
	tagCodeBlock  = "```"
	tagBlockQuote = "> "

	pipeBytes  = "|"
	spaceBytes = " "
)

type Text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Block struct {
	Type string `json:"type"`

	Text *Text `json:"text,omitempty"`
}

var escapes = [256][]byte{
	'&': []byte(`&amp;`),
	'<': []byte(`&lt;`),
	'>': []byte(`&gt;`),
}

func esc(w io.Writer, text []byte) {
	var start, end int
	for end < len(text) {
		if escSeq := escapes[text[end]]; escSeq != nil {
			w.Write(text[start:end])
			w.Write(escSeq)
			start = end + 1
		}
		end++
	}

	if start < len(text) && end <= len(text) {
		w.Write(text[start:end])
	}
}

func appendText(blocks []*Block, text string) []*Block {
	var last *Block
	if len(blocks) > 0 {
		last = blocks[len(blocks)-1]
	}

	if last != nil && last.Type == "section" {
		last.Text.Text += text
		return blocks
	}

	return append(blocks, &Block{
		Type: "section",
		Text: &Text{
			Type: textTypeMrkdwn,
			Text: text,
		},
	})
}

func main() {
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	var (
		blocks []*Block

		tbuf strings.Builder

		itemLevel        int
		itemOrderByLevel = map[int]int{}
	)

	parser := bf.New(bf.WithExtensions(bf.CommonExtensions))
	ast := parser.Parse([]byte(input))

	pp.Printf("ast:\n%s\n\n", ast)

	ast.Walk(func(node *bf.Node, entering bool) bf.WalkStatus {
		pp.Println(node.Type.String(), entering, string(node.Literal))

		switch node.Type {
		case bf.Document, bf.HTMLBlock, bf.HTMLSpan:
			break
		case bf.Table, bf.TableCell, bf.TableHead, bf.TableBody, bf.TableRow:
			break
		case bf.Image:
			break
		case bf.Emph:
			tbuf.WriteString(tagItalic)
		case bf.Strong:
			tbuf.WriteString(tagStrong)
		case bf.Del:
			tbuf.WriteString(tagStrike)
		case bf.Text:
			esc(&tbuf, node.Literal)
		case bf.Link:
			if entering {
				tbuf.WriteString(tagLink)
				if dest := node.LinkData.Destination; dest != nil {
					tbuf.Write(dest)
					tbuf.WriteString(pipeBytes)
				}
			} else {
				tbuf.WriteString(tagLinkClose)
			}
		case bf.Paragraph:
			if entering {
				if node.Parent.Type == bf.BlockQuote {
					tbuf.WriteString(tagBlockQuote)
				}
			} else {
				if node.Parent.Type == bf.BlockQuote {
					text := strings.Replace(tbuf.String(), "\n", "\n> ", -1)
					tbuf.Reset()
					tbuf.WriteString(text)
				}
				if node.Parent.Type != bf.Item {
					tbuf.WriteByte('\n')
				}
				tbuf.WriteByte('\n')
				blocks = appendText(blocks, tbuf.String())
				tbuf.Reset()
			}
		case bf.List:
			if entering {
				itemLevel++
				if node.ListFlags&bf.ListTypeOrdered != 0 {
					itemOrderByLevel[itemLevel] = 1
				}
			} else {
				delete(itemOrderByLevel, itemLevel)
				itemLevel--
			}
		case bf.Item:
			if entering {
				tbuf.WriteString(spaceBytes)
				for i := 1; i < itemLevel; i++ {
					tbuf.WriteString(spaceBytes)
					tbuf.WriteString(spaceBytes)
					tbuf.WriteString(spaceBytes)
				}
				if node.ListFlags&bf.ListTypeOrdered != 0 {
					tbuf.Write(append([]byte(strconv.Itoa(itemOrderByLevel[itemLevel])), node.ListData.Delimiter))
					itemOrderByLevel[itemLevel]++
				} else {
					tbuf.WriteString(tagItem)
				}
				tbuf.WriteString(spaceBytes)
			}

		case bf.Code:
			tbuf.WriteString(tagCode)
			esc(&tbuf, node.Literal)
			tbuf.WriteString(tagCode)

		case bf.CodeBlock:
			tbuf.WriteString(tagCodeBlock)
			esc(&tbuf, node.Literal)
			tbuf.WriteString(tagCodeBlock)

		case bf.Heading:
			if entering {
				esc(&tbuf, node.Literal)
			} else {
				blocks = append(blocks, &Block{
					Type: "header",
					Text: &Text{
						Type: textTypePlain,
						Text: tbuf.String(),
					},
				})
				blocks = append(blocks, &Block{
					Type: "divider",
				})
				tbuf.Reset()
			}

		case bf.HorizontalRule:
			blocks = append(blocks, &Block{
				Type: "divider",
			})

		case bf.BlockQuote:
			break

		default:
			panic("unknown node type " + node.Type.String())
		}

		return bf.GoToNext
	})

	pp.Printf("blocks:\n%v\n", blocks)

	json.NewEncoder(os.Stdout).Encode(struct {
		Blocks []*Block `json:"blocks"`
	}{
		Blocks: blocks,
	})
}
