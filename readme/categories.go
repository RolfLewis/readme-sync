package readme

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/xerrors"
)

type Category struct {
	Id    string `json:"_id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Order int    `json:"order"`
}

// TODO: add auto paging
func (c *Client) GetCategories(ctx context.Context) ([]Category, error) {
	url := "https://dash.readme.com/api/v1/categories"
	res, err := c.do(http.MethodGet, url, nil)
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
	url := fmt.Sprintf("https://dash.readme.com/api/v1/categories/%v", slug)
	res, err := c.do(http.MethodGet, url, nil)
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

func (c *Client) CreateCategory(ctx context.Context, title string) error {
	type createCategoryPayload struct {
		Title string `json:"title"`
		Type  string `json:"type"`
	}

	payload := createCategoryPayload{
		Title: title,
		Type:  "guide",
	}

	url := "https://dash.readme.com/api/v1/categories"
	res, err := c.do(http.MethodPost, url, payload)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}
	return nil
}
