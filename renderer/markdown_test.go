package renderer_test

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/yuin/goldmark/ast"
	goldParse "github.com/yuin/goldmark/parser"
	goldRend "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"golang.org/x/xerrors"

	"github.com/rolflewis/readme-sync/renderer"
)

var parser goldParse.Parser
var caller testRenderCaller

func init() {
	parser = goldParse.NewParser(
		goldParse.WithBlockParsers(goldParse.DefaultBlockParsers()...),
		goldParse.WithInlineParsers(goldParse.DefaultInlineParsers()...),
		goldParse.WithParagraphTransformers(goldParse.DefaultParagraphTransformers()...),
	)

	caller = testRenderCaller{
		funcs: make(map[ast.NodeKind]goldRend.NodeRendererFunc),
	}

	renderer.NewMarkdown().RegisterFuncs(caller)
}

func runTestCase(t *testing.T, in, want string) {
	inBytes := []byte(in)
	node := parser.Parse(text.NewReader(inBytes))

	got, err := caller.Render(inBytes, node)
	if err != nil {
		t.Fatal(err)
	}

	node.Dump(inBytes, 4)

	if got != want {
		node.Dump(inBytes, 4)
		t.Log("got:", got)
		t.Log("want:", want)
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestRenderer_SimpleHeaderWithText(t *testing.T) {
	in := "# heading 1\n\nthis is text\n"
	runTestCase(t, in, in) // expect no changes here
}

func TestRenderer_MultipleHeaders(t *testing.T) {
	in := "# heading 1\n## heading\n# heading1\n"
	runTestCase(t, in, in)
}

func TestRenderer_FixMissingEndLine(t *testing.T) {
	in := "# heading 1"
	out := "# heading 1\n"
	runTestCase(t, in, out)
}

func TestRenderer_Emphasis(t *testing.T) {
	in := "this is a *line* of text\n"
	runTestCase(t, in, in)
}

func TestRenderer_HeavyEmphasis(t *testing.T) {
	in := "this is a **line** of text\n"
	runTestCase(t, in, in)
}

func TestRenderer_BasicOrderedList(t *testing.T) {
	in := "1. element 1\n2. element 2\n3. element 3\n"
	runTestCase(t, in, in)
}

func TestRenderer_BasicOrderedListRenumber(t *testing.T) {
	in := "1. element 1\n1. element 2\n1. element 3\n"
	out := "1. element 1\n2. element 2\n3. element 3\n"
	runTestCase(t, in, out)
}

func TestRenderer_NestedOrderedList_Tabs(t *testing.T) {
	in := "1. element 1\n\t1. element 2\n\t\t1. element 3\n"
	out := "1. element 1\n   1. element 2\n      1. element 3\n"
	runTestCase(t, in, out)
}

func TestRenderer_NestedOrderedList_Spaces(t *testing.T) { // Special case: https://spec.commonmark.org/0.30/#example-312
	in := "1. element 1\n   1. element 2\n      1. element 3\n"
	out := "1. element 1\n   1. element 2\n      1. element 3\n"
	runTestCase(t, in, out)
}

func TestRenderer_BasicUnorderedList(t *testing.T) {
	in := "- element 1\n- element 2\n- element 3\n"
	runTestCase(t, in, in)
}

func TestRenderer_NestedUnorderedList_SmallIndent(t *testing.T) {
	in := "- element 1\n  - element 2\n    - element 3\n"
	runTestCase(t, in, in)
}

func TestRenderer_NestedUnorderedList_LargeIndent(t *testing.T) { // indent width is not preserved
	in := "- element 1\n    - element 2\n        - element 3\n"
	out := "- element 1\n  - element 2\n    - element 3\n"
	runTestCase(t, in, out)
}

func TestRenderer_BlockQuotes(t *testing.T) {
	in := "> quote 1\nquote 2\nquote 3\n" // parser is lazy: https://spec.commonmark.org/0.30/#example-233
	runTestCase(t, in, in)
}

func TestRenderer_Link(t *testing.T) {
	in := "[link label](guide1)\n"
	runTestCase(t, in, in)
}

func TestRenderer_ImageLink(t *testing.T) {
	in := "![link label](link_destination.png)\n"
	runTestCase(t, in, in)
}

type testRenderCaller struct {
	funcs map[ast.NodeKind]goldRend.NodeRendererFunc
}

func (rc testRenderCaller) Register(k ast.NodeKind, f goldRend.NodeRendererFunc) {
	rc.funcs[k] = f
}

func (rc testRenderCaller) Render(source []byte, n ast.Node) (string, error) {
	out := bytes.Buffer{}
	buf := bufio.NewWriter(&out)

	if err := ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		var err error
		s := ast.WalkStatus(ast.WalkContinue)
		if f := rc.funcs[n.Kind()]; f != nil {
			s, err = f(buf, source, n, entering)
			if err != nil {
				return s, xerrors.Errorf(": %w", err)
			}
		}
		return s, nil
	}); err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	buf.Flush()
	res, err := io.ReadAll(&out)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	return string(res), nil
}
