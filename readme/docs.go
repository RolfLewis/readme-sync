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
	Id       string `json:"_id,omitempty"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Excerpt  string `json:"excerpt"`
	Body     string `json:"body"`
	Category string `json:"categorySlug"`
	Parent   string `json:"parentDocSlug"`
	Hidden   bool   `json:"hidden"`
	Order    int    `json:"order"`
}

func (c *Client) PutDoc(ctx context.Context, doc Document) error {
	if _, err := do[Document](c, doOpts{
		method:         http.MethodPut,
		path:           fmt.Sprintf("/api/v1/docs/%v", doc.Slug),
		expectedStatus: http.StatusOK,
	}); err != nil {
		return xerrors.Errorf(": %w", err)
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
	type response struct {
		Document
		ParentId   string `json:"parentDoc"`
		CategoryId string `json:"category"`
	}

	resp, err := do[response](c, doOpts{
		method:         http.MethodGet,
		path:           fmt.Sprintf("/api/v1/docs/%v", slug),
		expectedStatus: http.StatusOK,
	})
	if err != nil {
		return Document{}, xerrors.Errorf(": %w", err)
	}

	if resp.Document == (Document{}) {
		return Document{}, nil
	}

	// TODO: this may burn API calls - good candidate for a preload/caching setup
	catSlug, err := c.getCategorySlugForId(ctx, resp.CategoryId)
	if err != nil {
		return Document{}, xerrors.Errorf(": %w", err)
	}
	resp.Category = catSlug

	if resp.ParentId != "" {
		// TODO: this may burn API calls - good candidate for a preload/caching setup
		parSlug, err := c.getDocSlugForId(ctx, catSlug, resp.ParentId)
		if err != nil {
			return Document{}, xerrors.Errorf(": %w", err)
		}
		resp.Parent = parSlug
	}

	return resp.Document, nil
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
	if _, err := do[Document](c, doOpts{
		method:         http.MethodPost,
		path:           "/api/v1/docs/",
		expectedStatus: http.StatusCreated,
		body:           doc,
	}); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}

func (c *Client) DeleteDoc(ctx context.Context, slug string) error {
	if _, err := do[Document](c, doOpts{
		method:         http.MethodDelete,
		path:           fmt.Sprintf("/api/v1/docs/%v", slug),
		expectedStatus: http.StatusNoContent,
	}); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}

// endpoint does not support paging at time of writing
func (c *Client) GetDocsForCategory(ctx context.Context, slug string) ([]Document, error) {
	type respDoc struct {
		Document
		Children []Document `json:"children"`
	}

	respList, err := do[[]respDoc](c, doOpts{
		method:         http.MethodGet,
		path:           fmt.Sprintf("/api/v1/categories/%v/docs", slug),
		expectedStatus: http.StatusOK,
	})
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	var docs []Document
	for _, resp := range respList {
		resp.Category = slug
		docs = append(docs, resp.Document)
		for _, child := range resp.Children {
			child.Category = slug
			child.Parent = resp.Slug
			docs = append(docs, child)
		}
	}

	return docs, nil
}
