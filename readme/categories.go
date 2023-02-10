package readme

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/xerrors"
)

type Category struct {
	Id    string `json:"_id,omitempty"`
	Slug  string `json:"slug,omitempty"`
	Title string `json:"title,omitempty"`
}

// TODO: add auto paging
func (c *Client) GetCategories(ctx context.Context) ([]Category, error) {
	path := "/api/v1/categories"
	cats, err := doAllPages[Category](c, http.MethodGet, path)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	return cats, nil
}

func (c *Client) GetCategory(ctx context.Context, slug string) (Category, error) {
	cat, err := do[Category](c, doOpts{
		method:         http.MethodGet,
		path:           fmt.Sprintf("/api/v1/categories/%v", slug),
		expectedStatus: http.StatusOK,
	})
	if err != nil {
		return Category{}, xerrors.Errorf(": %w", err)
	}
	return cat, nil
}

// Creates a category with the given name and a slug equivalent to the slugified name
// First creates the category with the slug title, then updates it to the original
// title if the title is not equal to the slug (change in case, spacing, etc)
// This is necessary due to lack of slug field on category creation.
func (c *Client) CreateCategory(ctx context.Context, cat Category) error {
	// First, create the category
	createPayload := Category{
		Title: cat.Slug,
	}

	if _, err := do[Category](c, doOpts{
		method:         http.MethodPost,
		path:           "/api/v1/categories",
		expectedStatus: http.StatusCreated,
		body:           createPayload,
	}); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	// Second, update it if the slug != title
	if cat.Slug != cat.Title {
		if err := c.UpdateCategory(ctx, cat); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	}

	return nil
}

func (c *Client) UpdateCategory(ctx context.Context, cat Category) error {
	updatePayload := Category{
		Title: cat.Title,
	}

	if _, err := do[Category](c, doOpts{
		method:         http.MethodPut,
		path:           fmt.Sprintf("/api/v1/categories/%v", cat.Slug),
		expectedStatus: http.StatusOK,
		body:           updatePayload,
	}); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}

func (c *Client) DeleteCategory(ctx context.Context, slug string) error {
	if _, err := do[Category](c, doOpts{
		method:         http.MethodDelete,
		path:           fmt.Sprintf("/api/v1/categories/%v", slug),
		expectedStatus: http.StatusNoContent,
	}); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}

func (c *Client) getCategorySlugForId(ctx context.Context, id string) (string, error) {
	// get doc's category slug.
	// as of writing, the readme api does not allow querying docs or cats by id, only slug.
	// however, the api does not allow you to get a doc's parent or category slugs directly, only id.
	// thus, this listing + match workaround is easiest workaround to implement this lookup
	cats, err := c.GetCategories(ctx)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	for _, cat := range cats {
		if cat.Id == id {
			return cat.Slug, nil
		}
	}
	return "", xerrors.New("no matching category found for id " + id)
}
