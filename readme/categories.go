package readme

import (
	"context"
	"encoding/json"
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
	res, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}

	var cats []Category
	if err := json.NewDecoder(res.Body).Decode(&cats); err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	return cats, nil
}

func (c *Client) GetCategory(ctx context.Context, slug string) (Category, error) {
	path := fmt.Sprintf("/api/v1/categories/%v", slug)
	res, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return Category{}, xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return Category{}, nil
	}

	if res.StatusCode != http.StatusOK {
		return Category{}, xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}

	var cat Category
	if err := json.NewDecoder(res.Body).Decode(&cat); err != nil {
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

	path := "/api/v1/categories"
	res, err := c.do(http.MethodPost, path, createPayload)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return xerrors.Errorf(": %w", handleErrorResponse(res.Body))
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

	path := fmt.Sprintf("/api/v1/categories/%v", cat.Slug)
	res, err := c.do(http.MethodPut, path, updatePayload)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}

	return nil
}

func (c *Client) DeleteCategory(ctx context.Context, slug string) error {
	path := fmt.Sprintf("/api/v1/categories/%v", slug)
	res, err := c.do(http.MethodDelete, path, nil)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		return xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}

	return nil
}
