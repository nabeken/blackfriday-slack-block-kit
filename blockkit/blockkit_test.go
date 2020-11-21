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
	assertText(t, "> hello\n> world\n> \n> abc\n> def\n\n", blocks[0])
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
