package docs

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-cmp/cmp"
	"github.com/rolflewis/readme-sync/readme"
	"golang.org/x/xerrors"
)

type CatMetadata struct {
	Title string
	Slug  string
}

// TODO: improve tracking of final category slug - may need to pass it back
func ProcessCategory(ctx context.Context, c *readme.Client, metadata CatMetadata) error {
	existing, err := c.GetCategory(ctx, metadata.Slug)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	cat := readme.Category{
		Title: metadata.Title,
		Slug:  metadata.Slug,
	}

	if existing == (readme.Category{}) {
		fmt.Printf("Creating category with slug \"%v\"\n", cat.Slug)
		if err := c.CreateCategory(ctx, cat); err != nil {
			return xerrors.Errorf(": %w", err)
		}
		return nil
	} else if cat.Id = existing.Id; existing != cat {
		log.Println(cmp.Diff(existing, cat))
		fmt.Printf("Updating category with slug \"%v\"\n", cat.Slug)
		if err := c.UpdateCategory(ctx, cat); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	} else {
		fmt.Printf("No change to category with slug \"%v\"\n", cat.Slug)
	}
	return nil
}
