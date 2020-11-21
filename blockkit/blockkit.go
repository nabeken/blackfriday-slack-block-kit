package blockkit

import (
	"bytes"
	"io"
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

// Layout represetns Slack Block Kit UI Layout. It can be used as a payload to Slack API.
type Layout struct {
	Blocks []*Block `json:"blocks"`
}

type Text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Block struct {
	Type string `json:"type"`
	Text *Text  `json:"text,omitempty"`
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

// Converter holds an internal state for converting blackfriday v2 AST into Slack Block Kit UI Framework.
type Converter struct {
	ast    *bf.Node
	buf    bytes.Buffer
	blocks []*Block

	// records level of ordered items per item level
	// Example:
	// 1.            # 0 -> 1
	// 2.            # 0 -> 2
	//    1.         # 1 -> 1
	//    2.         # 1 -> 2
	//       1.      # 2 -> 1
	//       2.      # 2 -> 2
	itemLevel        int
	itemOrderByLevel map[int]int

	debug bool
}

func NewConverter(ast *bf.Node) *Converter {
	return &Converter{
		ast:              ast,
		itemOrderByLevel: make(map[int]int),
	}
}

// Debug togles a debug flag.
func (c *Converter) Debug() *Converter {
	c.debug = !c.debug
	return c
}

func (c *Converter) Convert() Layout {
	if c.debug {
		pp.Printf("ast:\n%s\n\n", c.ast)
	}
	c.ast.Walk(func(node *bf.Node, entering bool) bf.WalkStatus {
		if c.debug {
			pp.Println(node.Type.String(), entering, string(node.Literal))
		}

		switch node.Type {
		case bf.Document, bf.HTMLBlock, bf.HTMLSpan:
			break
		case bf.Table, bf.TableCell, bf.TableHead, bf.TableBody, bf.TableRow:
			break
		case bf.Image:
			break
		case bf.Emph:
			c.buf.WriteString(tagItalic)
		case bf.Strong:
			c.buf.WriteString(tagStrong)
		case bf.Del:
			c.buf.WriteString(tagStrike)
		case bf.Text:
			esc(&c.buf, node.Literal)
		case bf.Link:
			if entering {
				c.buf.WriteString(tagLink)
				if dest := node.LinkData.Destination; dest != nil {
					c.buf.Write(dest)
					c.buf.WriteString(pipeBytes)
				}
			} else {
				c.buf.WriteString(tagLinkClose)
			}
		case bf.Paragraph:
			if !entering {
				if node.Parent.Type != bf.Item {
					c.buf.WriteByte('\n')
				}

				c.buf.WriteByte('\n')

				if node.Parent.Type != bf.BlockQuote {
					c.blocks = appendText(c.blocks, c.buf.String())
					c.buf.Reset()
				}
			}
		case bf.List:
			if entering {
				c.itemLevel++
				if node.ListFlags&bf.ListTypeOrdered != 0 {
					c.itemOrderByLevel[c.itemLevel] = 1
				}
			} else {
				delete(c.itemOrderByLevel, c.itemLevel)
				c.itemLevel--
			}
		case bf.Item:
			if entering {
				c.buf.WriteString(spaceBytes)
				for i := 1; i < c.itemLevel; i++ {
					c.buf.WriteString(spaceBytes)
					c.buf.WriteString(spaceBytes)
					c.buf.WriteString(spaceBytes)
				}
				if node.ListFlags&bf.ListTypeOrdered != 0 {
					c.buf.Write(append([]byte(strconv.Itoa(c.itemOrderByLevel[c.itemLevel])), node.ListData.Delimiter))
					c.itemOrderByLevel[c.itemLevel]++
				} else {
					c.buf.WriteString(tagItem)
				}
				c.buf.WriteString(spaceBytes)
			}

		case bf.Code:
			c.buf.WriteString(tagCode)
			esc(&c.buf, node.Literal)
			c.buf.WriteString(tagCode)

		case bf.CodeBlock:
			c.buf.WriteString(tagCodeBlock)
			esc(&c.buf, node.Literal)
			c.buf.WriteString(tagCodeBlock)

		case bf.Heading:
			if entering {
				esc(&c.buf, node.Literal)
			} else {
				c.blocks = append(c.blocks, &Block{
					Type: "header",
					Text: &Text{
						Type: textTypePlain,
						Text: c.buf.String(),
					},
				})
				c.blocks = append(c.blocks, &Block{
					Type: "divider",
				})
				c.buf.Reset()
			}

		case bf.HorizontalRule:
			c.blocks = append(c.blocks, &Block{
				Type: "divider",
			})

		case bf.BlockQuote:
			if entering {
				c.buf.WriteString(tagBlockQuote)
			} else {
				text := strings.Replace(c.buf.String(), "\n", "\n> ", -1)
				c.blocks = appendText(c.blocks, text)
				c.buf.Reset()
			}

		default:
			panic("unknown node type " + node.Type.String())
		}

		return bf.GoToNext
	})

	layout := Layout{
		Blocks: c.blocks,
	}

	if c.debug {
		pp.Printf("layout:\n%v\n", layout)
	}

	return layout
}

// Convert converts blackfriday v2 AST into Layout.
func Convert(ast *bf.Node) Layout {
	return NewConverter(ast).Convert()
}
