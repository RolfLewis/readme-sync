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

	// taken from https://github.com/yuin/goldmark/blob/master/extension/table.go
	md.Parser().AddOptions(
		parser.WithParagraphTransformers(
			util.Prioritized(extension.NewTableParagraphTransformer(), 200),
		),
		parser.WithASTTransformers(
			util.Prioritized(extension.NewTableASTTransformer(), 0),
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
