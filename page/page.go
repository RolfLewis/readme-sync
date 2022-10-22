package page

import (
	"context"
	"log"
	"os"

	"github.com/rolflewis/readme-sync/renderer"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	gren "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
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
			extension.GFM,
			meta.New(
				meta.WithStoresInDocument(),
			),
		),
		goldmark.WithRenderer(
			gren.NewRenderer(gren.WithNodeRenderers(util.Prioritized(renderer.NewRenderer(), 1000))),
		),
	)

	document := md.Parser().Parse(text.NewReader(source))
	metadata := document.OwnerDocument().Meta()
	log.Println(metadata)

	document.Dump(source, 2)
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
