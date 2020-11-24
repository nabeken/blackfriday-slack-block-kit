package blockkit

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/k0kubun/pp"
	bf "github.com/russross/blackfriday/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockQuoteConvert(t *testing.T) {
	_, blocks := newConv(string(mustOpenTestData("example_blockquote.md")))
	assertText(t, "> hello\n> world\n>\n> abc\n> def\n\n", blocks[0])
}

func TestListParagraphConvert(t *testing.T) {
	_, blocks := newConv(string(mustOpenTestData("example_list.md")))
	assertText(t, "• abc\n• def\n   • foobar\n   • moge\n\nfoobar\n\n", blocks[0])
}

func TestCodeBlockConvert(t *testing.T) {
	_, blocks := newConv(string(mustOpenTestData("example_codeblock.md")))
	assertText(t, "```\ngo func() {\n  println(\"hello world\")\n}()\n```\n\n", blocks[0])
}

func TestFullExample(t *testing.T) {
	expected := []*Block{
		&Block{
			Type: "header",
			Text: newPlainText("Heading 1"),
		},
		&Block{
			Type: "divider",
		},
		&Block{
			Type: "header",
			Text: newPlainText("Heading 2"),
		},
		&Block{
			Type: "divider",
		},
		&Block{
			Type: "header",
			Text: newPlainText("Heading 3"),
		},
		&Block{
			Type: "divider",
		},
		&Block{
			Type: "header",
			Text: newPlainText("Heading 4"),
		},
		&Block{
			Type: "divider",
		},
		&Block{
			Type: "header",
			Text: newPlainText("Heading 5"),
		},
		&Block{
			Type: "divider",
		},
		&Block{
			Type: "header",
			Text: newPlainText("Heading 6"),
		},
		&Block{
			Type: "divider",
		},
		&Block{
			Type: "section",
			Text: &Text{
				Type: textTypeMrkdwn,
				Text: "Hello, ~Markdown~ *mrkdwn*!\n\n`mrkdwn` is text formatting markup style in <https://slack.com/|Slack>.\n\n",
			},
		},
		&Block{
			Type: "divider",
		},
		&Block{
			Type: "section",
			Text: &Text{
				Type: textTypeMrkdwn,
				Text: "• First\n• Second\n   • Sub item 1\n   • Sub item 2\n      • Sub Sub A\n      • Sub Sub B\n   • C\n   • <https://slack.com|Slack>\n• Third\n   1. Ordered list 1\n   2. Ordered list 2\n   3. Ordered list 3\n      1. Ordered sub list 1\n      2. Ordered sub list 2\n      3. Ordered sub list 3\n   4. Ordered list 4\n> _This is blockquote._\n> <https://slack.com|Slack>\n>\n> *This is the second paragraph in blockquote.*\n> <https://slack.com|Slack>\n\n```\nconsole.log('Hello, mrkdwn!')\n```\n\nThis is the last paragraph.\n\n",
			},
		},
	}
	_, blocks := newConv(string(mustOpenTestData("full_example.md")))
	assert.Equal(t, expected, blocks)
}

func TestSimpleConvert(t *testing.T) {
	type testCase struct {
		input    string
		expected string
	}
	for n, tc := range map[string]testCase{
		"text": testCase{
			input:    "abc",
			expected: "abc\n\n",
		},
		"italic": testCase{
			input:    "*abc*",
			expected: "_abc_\n\n",
		},
		"bold": testCase{
			input:    "**abc**",
			expected: "*abc*\n\n",
		},
		"bold and italic": testCase{
			input:    "***abc***",
			expected: "*_abc_*\n\n",
		},
		"link": testCase{
			input:    "[Slack](https://slack.com)",
			expected: "<https://slack.com|Slack>\n\n",
		},
		"code": testCase{
			input:    "`code`",
			expected: "`code`\n\n",
		},
		"paragraph": testCase{
			input:    "abc\ndef",
			expected: "abc\ndef\n\n",
		},
	} {
		_, blocks := newConv(tc.input)
		require.Len(t, blocks, 1, n)

		if !assertText(t, tc.expected, blocks[0]) {
			pp.Printf("blocks:\n%s\n\n", blocks)
		}
	}
}

func mustOpenTestData(fn string) []byte {
	data, err := ioutil.ReadFile("_testdata/" + fn)
	if err != nil {
		panic(fmt.Sprintf("failed to read _testdata/%s: %s", fn, err))
	}
	return data
}

func assertText(t *testing.T, expected string, block *Block) bool {
	assert.Equal(t, "section", block.Type)
	assert.Equal(t, "mrkdwn", block.Text.Type)
	return assert.Equal(t, expected, block.Text.Text)
}

func newConv(input string) (*bf.Node, []*Block) {
	parser := bf.New(bf.WithExtensions(bf.CommonExtensions))
	ast := parser.Parse([]byte(input))
	conv := NewConverter(ast)
	if testing.Verbose() {
		conv.Debug()
	}
	return ast, conv.Convert().Blocks
}

func newPlainText(text string) *Text {
	return &Text{
		Type: textTypePlain,
		Text: text,
	}
}
