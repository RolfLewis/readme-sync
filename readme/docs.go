package readme

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/xerrors"
)

type apiErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type Document struct {
	Slug     string `json:"-"`
	Title    string `json:"title"`
	Excerpt  string `json:"excerpt"`
	Body     string `json:"body"`
	Category string `json:"categorySlug"`
	Parent   string `json:"parentDocSlug"`
	Hidden   bool   `json:"hidden"`
	Order    int    `json:"order"`
}

func (c *Client) PutDoc(ctx context.Context, doc Document) error {
	url := fmt.Sprintf("https://dash.readme.com/api/v1/docs/%v", doc.Slug)
	res, err := c.do(http.MethodPut, url, doc)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}

	return nil
}

func handleErrorResponse(body io.Reader) error {
	var apiErr apiErrorResponse
	if err := json.NewDecoder(body).Decode(&apiErr); err != nil {
		return xerrors.Errorf(": %w", err)
	}
	return xerrors.New(apiErr.Error + ":" + apiErr.Message)
}

func (c *Client) GetDoc(ctx context.Context, slug string) (Document, error) {
	url := fmt.Sprintf("https://dash.readme.com/api/v1/docs/%v", slug)

	res, err := c.do(http.MethodGet, url, nil)
	if err != nil {
		return Document{}, xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return Document{}, nil
	}

	if res.StatusCode != http.StatusOK {
		return Document{}, xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}

	var response struct {
		Title     string `json:"title"`
		Excerpt   string `json:"excerpt"`
		Type      string `json:"type"`
		Body      string `json:"body"`
		ParentDoc string `json:"parentDoc"`
		Category  string `json:"category"`
		Hidden    bool   `json:"hidden"`
		Order     int    `json:"order"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return Document{}, xerrors.Errorf(": %w", err)
	}

	retDoc := Document{
		Slug:    slug,
		Title:   response.Title,
		Excerpt: response.Excerpt,
		Order:   response.Order,
		Hidden:  response.Hidden,
		Body:    response.Body,
	}

	// TODO: this may burn API calls - good candidate for a preload/caching setup
	catSlug, err := c.getCategorySlugForId(ctx, response.Category)
	if err != nil {
		return Document{}, xerrors.Errorf(": %w", err)
	}
	retDoc.Category = catSlug

	if response.ParentDoc != "" {
		// TODO: this may burn API calls - good candidate for a preload/caching setup
		parSlug, err := c.getDocSlugForId(ctx, catSlug, response.ParentDoc)
		if err != nil {
			return Document{}, xerrors.Errorf(": %w", err)
		}
		retDoc.Parent = parSlug
	}

	return retDoc, nil
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
	return "", xerrors.New("no matching category found for id")
}

func (c *Client) getDocSlugForId(ctx context.Context, catSlug, docId string) (string, error) {
	// get doc's slug.
	// as of writing, the readme api does not allow querying docs or cats by id, only slug.
	// however, the api does not allow you to get a doc's parent or category slugs directly, only id.
	// thus, this listing + match workaround is easiest workaround to implement this lookup
	docs, err := c.GetDocsForCategory(ctx, catSlug)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	for _, doc := range docs {
		if doc.Id == docId {
			return doc.Slug, nil
		}
	}
	return "", xerrors.New("no matching doc found for id")
}

func (c *Client) CreateDoc(ctx context.Context, doc Document) error {
	url := "https://dash.readme.com/api/v1/docs/"
	res, err := c.do(http.MethodPost, url, doc)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}

	return nil
}

type objectStub struct {
	Id   string `json:"_id"`
	Slug string `json:"slug"`
}

// TODO: add auto paging
func (c *Client) GetDocsForCategory(ctx context.Context, slug string) ([]objectStub, error) {
	url := fmt.Sprintf("https://dash.readme.com/api/v1/categories/%v/docs", slug)
	res, err := c.do(http.MethodGet, url, nil)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}

	var stubs []objectStub
	if err := json.NewDecoder(res.Body).Decode(&stubs); err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}
	return stubs, nil
}
