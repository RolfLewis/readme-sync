package page

import (
	"context"
	"os"

	"github.com/rolflewis/readme-sync/renderer"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gren "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"golang.org/x/xerrors"
)

func ProcessPage(ctx context.Context, path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			// extension.GFM,
			meta.New(
				meta.WithStoresInDocument(),
			),
		),
		goldmark.WithRenderer(
			gren.NewRenderer(gren.WithNodeRenderers(util.Prioritized(renderer.NewMarkdown(), 1000))),
		),
	)

	md.Parser().AddOptions(
		parser.WithParagraphTransformers(
			// https://github.com/yuin/goldmark/blob/c71a97b8372876d63528b54cedecf1104530fe3b/extension/table.go#L542
			util.Prioritized(extension.NewTableParagraphTransformer(), 200),
		),
		parser.WithASTTransformers(
			// https://github.com/yuin/goldmark/blob/c71a97b8372876d63528b54cedecf1104530fe3b/extension/table.go#L542
			util.Prioritized(extension.NewTableASTTransformer(), 0),
		),
		parser.WithInlineParsers(
			// https://github.com/yuin/goldmark/blob/c71a97b8372876d63528b54cedecf1104530fe3b/extension/strikethrough.go#L110
			util.Prioritized(extension.NewStrikethroughParser(), 500),
			// https://github.com/yuin/goldmark/blob/c71a97b8372876d63528b54cedecf1104530fe3b/extension/tasklist.go#L109
			util.Prioritized(extension.NewTaskCheckBoxParser(), 0),
		),
	)

	// document := md.Parser().Parse(text.NewReader(source))
	// metadata := document.OwnerDocument().Meta()
	// log.Println(metadata)

	// document.Dump(source, 2)
	file, err := os.OpenFile("test.md", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer file.Close()

	if err := md.Convert(source, file); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}
