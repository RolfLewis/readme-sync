package docs

import (
	"context"
	"log"

	"github.com/rolflewis/readme-sync/readme"
	"golang.org/x/xerrors"
)

// TODO: improve tracking of final category slug - may need to pass it back
func ProcessCategory(ctx context.Context, c *readme.Client, slug string) error {
	existing, err := c.GetCategory(ctx, slug)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	if existing == (readme.Category{}) {
		log.Println("create category")
		if err := c.CreateCategory(ctx, slug); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	} else {
		log.Println("category exists")
	}
	// TODO: Add handling for Category ordering
	return nil
}
