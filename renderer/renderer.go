package renderer

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	goldmark "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"golang.org/x/xerrors"
)

var indent = []byte("    ")

type renderer struct {
	config *goldmark.Config

	quoteLevel int

	listLevel        int
	nextListPosition int
	nextListMarker   byte
	nextListOffset   int
}

func NewRenderer(options ...goldmark.Option) goldmark.NodeRenderer {
	r := &renderer{
		config:     goldmark.NewConfig(),
		listLevel:  -1,
		quoteLevel: -1,
	}

	// ignore options until needed
	// for _, opt := range options {
	// 	opt.SetHTMLOption(&r.config)
	// }
	return r
}

func (r *renderer) RegisterFuncs(reg goldmark.NodeRendererFuncRegisterer) {
	// default registrations
	// blocks
	reg.Register(ast.KindDocument, r.renderDocument)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindTextBlock, r.renderTextBlock)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)

	// inlines
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindString, r.renderText)

	// GFM Extensions
	// Tables
	// reg.Register(exast.KindTable, r.renderTable)
	// reg.Register(exast.KindTableHeader, r.renderTableHeader)
	// reg.Register(exast.KindTableRow, r.renderTableRow)
	// reg.Register(exast.KindTableCell, r.renderTableCell)
	// // Strikethrough
	// reg.Register(exast.KindStrikethrough, r.renderStrikethrough)
	// // Checkbox
	// reg.Register(exast.KindTaskCheckBox, r.renderTaskCheckBox)
}

func preRender(w util.BufWriter, source []byte, node ast.Node, entering bool) error {
	if entering && node.Type() == ast.TypeBlock && node.HasBlankPreviousLines() {
		if err := w.WriteByte('\n'); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	}
	return nil
}

func (r *renderer) writeBlockQuotePrefix(w util.BufWriter) error {
	for i := -1; i < r.quoteLevel; i++ {
		if _, err := w.WriteString("> "); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	}
	return nil
}

func (r *renderer) writeListItemPrefix(w util.BufWriter) error {
	if _, err := w.Write(bytes.Repeat(indent, r.listLevel)); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	numString := strconv.Itoa(r.nextListPosition)
	if _, err := w.WriteString(numString); err != nil {
		return xerrors.Errorf(": %w", err)
	}
	r.nextListPosition++

	if err := w.WriteByte(r.nextListMarker); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	// account for marker in offset
	if _, err := w.Write(bytes.Repeat([]byte{' '}, r.nextListOffset-1)); err != nil {
		return xerrors.Errorf(": %w", err)
	}
	return nil
}

func (r *renderer) lineWriterHelper(w util.BufWriter, source []byte, n ast.Node) error {
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		if err := r.writeBlockQuotePrefix(w); err != nil {
			return xerrors.Errorf(": %w", err)
		}

		if r.listLevel != -1 {
			if err := r.writeListItemPrefix(w); err != nil {
				return xerrors.Errorf(": %w", err)
			}
		}

		line := n.Lines().At(i)
		if _, err := w.Write(line.Value(source)); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	}
	return nil
}

func linkPrinterHelper(w util.BufWriter, url, label []byte) error {
	filled := fmt.Sprintf("[%s](%s)", util.EscapeHTML(label), util.EscapeHTML(util.URLEscape(url, false)))
	if _, err := w.WriteString(filled); err != nil {
		return xerrors.Errorf(": %w", err)
	}
	return nil
}

func (r *renderer) renderDocument(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *renderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	n := node.(*ast.Heading)
	if entering {
		// print prefix
		prefix := fmt.Sprintf("%s ", strings.Repeat("#", n.Level))
		if _, err := w.WriteString(prefix); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	} else {
		// add newline
		if err := w.WriteByte('\n'); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if entering {
		r.quoteLevel++
		return ast.WalkContinue, nil
	} else {
		r.quoteLevel--
		// add newline
		if err := w.WriteByte('\n'); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if entering {
		if _, err := w.WriteString("```\n"); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
		if err := r.lineWriterHelper(w, source, node); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	} else {
		// add close and newline
		if _, err := w.WriteString("```\n"); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	n := node.(*ast.FencedCodeBlock)
	if entering {
		prefix := fmt.Sprintf("```%s\n", n.Language(source))
		if _, err := w.WriteString(prefix); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
		if err := r.lineWriterHelper(w, source, node); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	} else {
		// add close and newline
		if _, err := w.WriteString("```\n"); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	n := node.(*ast.HTMLBlock)
	if entering {
		l := n.Lines().Len()
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			w.Write(line.Value(source))
		}
	} else {
		if n.HasClosure() {
			closure := n.ClosureLine
			w.Write(closure.Value(source))
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if entering {
		n := node.(*ast.List)
		r.nextListPosition = n.Start
		r.nextListMarker = n.Marker
		r.listLevel++
	} else {
		r.listLevel--
	}

	return ast.WalkContinue, nil
}

func (r *renderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if entering {
		n := node.(*ast.ListItem)
		r.nextListOffset = n.Offset

		// if fc := node.FirstChild(); fc != nil {
		// 	if fc.Kind() != ast.KindTextBlock {
		// 		// add newline
		// 		if err := w.WriteByte('\n'); err != nil {
		// 			return ast.WalkStop, xerrors.Errorf(": %w", err)
		// 		}
		// 	}
		// }
	} else {
		r.nextListOffset = 0
		// add newline
		if err := w.WriteByte('\n'); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if !entering {
		// add newline
		if err := w.WriteByte('\n'); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderTextBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if !entering {
		if _, ok := node.NextSibling().(ast.Node); ok && node.FirstChild() != nil {
			// add newline
			if err := w.WriteByte('\n'); err != nil {
				return ast.WalkStop, xerrors.Errorf(": %w", err)
			}
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if entering {
		if _, err := w.WriteString("---"); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	n := node.(*ast.AutoLink)
	if !entering {
		return ast.WalkContinue, nil
	}

	url := n.URL(source)
	label := n.Label(source)

	if n.AutoLinkType == ast.AutoLinkEmail && !bytes.HasPrefix(bytes.ToLower(url), []byte("mailto:")) {
		prefix := []byte("mailto:")
		url = bytes.Join([][]byte{
			prefix,
			url,
		}, nil)
	}

	if err := linkPrinterHelper(w, url, label); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	return ast.WalkContinue, nil
}

func (r *renderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if entering {
		if err := w.WriteByte('`'); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			segment := c.(*ast.Text).Segment
			value := segment.Value(source)
			if bytes.HasSuffix(value, []byte("\n")) {
				if _, err := w.Write(value[:len(value)-1]); err != nil {
					return ast.WalkStop, xerrors.Errorf(": %w", err)
				}
				if c != node.LastChild() {
					if err := w.WriteByte(' '); err != nil {
						return ast.WalkStop, xerrors.Errorf(": %w", err)
					}
				}
			} else {
				if _, err := w.Write(value); err != nil {
					return ast.WalkStop, xerrors.Errorf(": %w", err)
				}
			}
		}
		return ast.WalkSkipChildren, nil
	}
	if err := w.WriteByte('`'); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	n := node.(*ast.Emphasis)
	marker := strings.Repeat("*", n.Level)
	if _, err := w.WriteString(marker); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	status, err := r.renderAutoLink(w, source, node, entering)
	if err != nil {
		return status, xerrors.Errorf(": %w", err)
	}
	return status, nil
}

func (r *renderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Image)
	url := n.Destination
	alt := n.Text(source)

	if err := w.WriteByte('!'); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if err := linkPrinterHelper(w, url, alt); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	return ast.WalkSkipChildren, nil
}

func (r *renderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if entering {
		n := node.(*ast.RawHTML)
		l := n.Segments.Len()
		for i := 0; i < l; i++ {
			segment := n.Segments.At(i)
			_, _ = w.Write(segment.Value(source))
		}
	}
	return ast.WalkSkipChildren, nil
}

func (r *renderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if entering {
		n := node.(*ast.Text)
		segment := n.Segment

		log.Println(segment, string(segment.Value(source)), string(n.Text(source)))

		if _, err := w.Write(segment.Value(source)); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}

		if n.HardLineBreak() || n.SoftLineBreak() {
			if err := w.WriteByte('\n'); err != nil {
				return ast.WalkStop, xerrors.Errorf(": %w", err)
			}
		}
	}
	return ast.WalkContinue, nil
}

func (r *renderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if err := preRender(w, source, node, entering); err != nil {
		return ast.WalkStop, xerrors.Errorf(": %w", err)
	}

	if entering {
		n := node.(*ast.String)
		if _, err := w.Write(n.Value); err != nil {
			return ast.WalkStop, xerrors.Errorf(": %w", err)
		}
	}
	return ast.WalkContinue, nil
}
