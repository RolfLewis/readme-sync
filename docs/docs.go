package docs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/gosimple/slug"
	"github.com/rolflewis/readme-sync/readme"
	"golang.org/x/xerrors"
)

type DocMetadata struct {
	Category string
	Parent   string
	Slug     string
	Filepath string
}

type Catalog struct {
	Categories map[string]struct{}
	Docs       map[string]DocMetadata
}

func WalkCatalog(ctx context.Context, docsPath string) (Catalog, error) {
	catalog := Catalog{
		Categories: make(map[string]struct{}),
		Docs:       make(map[string]DocMetadata),
	}

	cats, err := os.ReadDir(docsPath)
	if err != nil {
		return Catalog{}, xerrors.Errorf(": %w", err)
	}

	for _, cat := range cats {
		if !cat.IsDir() {
			return Catalog{}, xerrors.New("found non-dir in categories layer")
		}
		categorySlug := slug.Make(cat.Name())
		if categorySlug != cat.Name() {
			return Catalog{}, xerrors.New("category folder name not slugified - should be named " + categorySlug)
		}
		if _, dup := catalog.Categories[categorySlug]; dup {
			return Catalog{}, xerrors.New("duplicate category slug detected")
		}
		catalog.Categories[categorySlug] = struct{}{}

		catPath := fmt.Sprintf("%v%v%v", docsPath, string(os.PathSeparator), cat.Name())
		catContents, err := os.ReadDir(catPath)
		if err != nil {
			return Catalog{}, xerrors.Errorf(": %w", err)
		}

		for _, cc := range catContents {
			if !cc.IsDir() { // doc with no parent
				slug := slug.Make(strings.TrimSuffix(cc.Name(), filepath.Ext(cc.Name())))
				if _, dup := catalog.Docs[slug]; dup {
					return Catalog{}, xerrors.New("duplicate doc slug")
				}
				catalog.Docs[slug] = DocMetadata{
					Category: categorySlug,
					Slug:     slug,
					Filepath: fmt.Sprintf("%v%v%v", catPath, string(os.PathSeparator), cc.Name()),
				}
			} else { // doc with a parent
				folderPath := fmt.Sprintf("%v%v%v", catPath, string(os.PathSeparator), cc.Name())
				foldContents, err := os.ReadDir(folderPath)
				if err != nil {
					return Catalog{}, xerrors.Errorf(": %w", err)
				}

				var foundFolderPage bool // need a doc with same slug as folder inside folder
				folderSlug := slug.Make(cc.Name())
				for _, fc := range foldContents {
					if fc.IsDir() {
						return Catalog{}, xerrors.New("nested too deep")
					}

					slug := strings.TrimSuffix(fc.Name(), filepath.Ext(fc.Name()))
					if _, dup := catalog.Docs[slug]; dup {
						return Catalog{}, xerrors.New("duplicate doc slug")
					}

					meta := DocMetadata{
						Category: categorySlug,
						Slug:     slug,
						Filepath: fmt.Sprintf("%v%v%v", folderPath, string(os.PathSeparator), fc.Name()),
					}

					if folderSlug == slug {
						foundFolderPage = true
					} else {
						meta.Parent = folderSlug
					}

					catalog.Docs[slug] = meta
				}

				if !foundFolderPage {
					return Catalog{}, xerrors.New("no folder page found")
				}
			}
		}
	}

	return catalog, nil
}

type docFrontMatter struct {
	Title   string `yaml:"title"`
	Excerpt string `yaml:"excerpt"`
	Order   int    `yaml:"order"`
	Hidden  bool   `yaml:"hidden"`
}

func (fm *docFrontMatter) validate() error {
	if fm.Title == "" {
		return xerrors.New("title is required")
	}
	if fm.Order > 999 || fm.Order < 0 {
		return xerrors.New("order must be between 0 and 999 inclusive")
	}
	return nil
}

func ProcessDoc(ctx context.Context, c *readme.Client, metadata DocMetadata) error {
	f, err := os.Open(metadata.Filepath)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	var matter docFrontMatter
	rest, err := frontmatter.MustParse(f, &matter)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	if err := matter.validate(); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	document := readme.Document{
		Category: metadata.Category,
		Parent:   metadata.Parent,
		Slug:     metadata.Slug,
		Title:    matter.Title,
		Excerpt:  matter.Excerpt,
		Order:    matter.Order,
		Hidden:   matter.Hidden,
		Body:     strings.TrimSpace(string(rest)), // readme cleans whitespace
	}

	existing, err := c.GetDoc(ctx, document.Slug)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	if existing == (readme.Document{}) {
		fmt.Printf("Creating Doc with Slug %v\n", document.Slug)
		if err := c.CreateDoc(ctx, document); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	} else if existing != document {
		fmt.Printf("Updating Doc with Slug %v\n", document.Slug)
		if err := c.PutDoc(ctx, document); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	} else {
		fmt.Printf("No Change to Doc with Slug %v\n", document.Slug)
	}
	return nil
}
