package confluence

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ContentList is a paginated list of content (child pages, space content).
type ContentList struct {
	Results []Content `json:"results"`
	Size    int       `json:"size"`
	Limit   int       `json:"limit"`
	Start   int       `json:"start"`
}

// SpaceList is a paginated list of spaces.
type SpaceList struct {
	Results []Space `json:"results"`
	Size    int     `json:"size"`
	Limit   int     `json:"limit"`
	Start   int     `json:"start"`
}

// ListSpaces returns spaces, optionally filtered by type ("global"|"personal").
func (c *Client) ListSpaces(ctx context.Context, spaceType string, limit, start int) (*SpaceList, error) {
	q := url.Values{}
	if spaceType != "" {
		q.Set("type", spaceType)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if start > 0 {
		q.Set("start", strconv.Itoa(start))
	}
	var out SpaceList
	if err := c.doJSON(ctx, "GET", "/rest/api/space", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSpace fetches a single space (use expand "homepage" to get its root page).
func (c *Client) GetSpace(ctx context.Context, key string, expand []string) (*Space, error) {
	q := url.Values{}
	if len(expand) > 0 {
		q.Set("expand", strings.Join(expand, ","))
	}
	var out Space
	if err := c.doJSON(ctx, "GET", "/rest/api/space/"+url.PathEscape(key), q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetChildPages returns the direct child pages of a page.
func (c *Client) GetChildPages(ctx context.Context, id string, limit, start int) (*ContentList, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if start > 0 {
		q.Set("start", strconv.Itoa(start))
	}
	var out ContentList
	if err := c.doJSON(ctx, "GET", "/rest/api/content/"+url.PathEscape(id)+"/child/page", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSpaceRootPages returns the top-level (depth=root) pages of a space.
func (c *Client) GetSpaceRootPages(ctx context.Context, key string, limit int) (*ContentList, error) {
	q := url.Values{}
	q.Set("depth", "root")
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var out ContentList
	if err := c.doJSON(ctx, "GET", "/rest/api/space/"+url.PathEscape(key)+"/content/page", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MovePage re-parents a page by updating its ancestors — the Server/Data
// Center compatible way to move pages in the hierarchy (the dedicated
// /content/{id}/move endpoint only exists on Cloud and newer DC versions).
//
// newParentID sets the new parent. This covers promotion (re-parent to a
// grandparent), demotion (re-parent under a sibling), and general re-parenting.
// The current title and body are preserved and the version is auto-incremented.
//
// An empty newParentID sends empty ancestors to request a move to the space's
// top level, but many Server/DC versions ignore this — moving a page to the
// very top level is generally not supported via REST there.
func (c *Client) MovePage(ctx context.Context, id, newParentID string) (*Content, error) {
	current, err := c.Get(ctx, id, []string{"version", "body.storage", "space"})
	if err != nil {
		return nil, fmt.Errorf("fetching page to move: %w", err)
	}
	if current.Version == nil {
		return nil, fmt.Errorf("could not determine current version of %s", id)
	}

	payload := map[string]any{
		"id":      id,
		"type":    current.Type,
		"title":   current.Title,
		"version": map[string]int{"number": current.Version.Number + 1},
	}
	if newParentID != "" {
		payload["ancestors"] = []map[string]string{{"id": newParentID}}
	} else {
		payload["ancestors"] = []any{} // empty -> top level of the space
	}
	// Preserve the existing body; some Confluence versions require it on update.
	if current.Body != nil && current.Body.Storage != nil {
		payload["body"] = map[string]any{
			"storage": map[string]string{
				"value":          current.Body.Storage.Value,
				"representation": "storage",
			},
		}
	}

	var out Content
	if err := c.doJSON(ctx, "PUT", "/rest/api/content/"+url.PathEscape(id), nil, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletePage deletes (trashes) a page by ID.
func (c *Client) DeletePage(ctx context.Context, id string) error {
	return c.doJSON(ctx, "DELETE", "/rest/api/content/"+url.PathEscape(id), nil, nil, nil)
}

// Label is a content label.
type Label struct {
	Prefix string `json:"prefix,omitempty"`
	Name   string `json:"name"`
	ID     string `json:"id,omitempty"`
}

// LabelList is the response from listing labels.
type LabelList struct {
	Results []Label `json:"results"`
	Size    int     `json:"size"`
}

// GetLabels lists the labels on a piece of content.
func (c *Client) GetLabels(ctx context.Context, id string) (*LabelList, error) {
	var out LabelList
	if err := c.doJSON(ctx, "GET", "/rest/api/content/"+url.PathEscape(id)+"/label", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddLabels adds one or more labels to a piece of content.
func (c *Client) AddLabels(ctx context.Context, id string, names []string) (*LabelList, error) {
	body := make([]Label, 0, len(names))
	for _, n := range names {
		body = append(body, Label{Prefix: "global", Name: n})
	}
	var out LabelList
	if err := c.doJSON(ctx, "POST", "/rest/api/content/"+url.PathEscape(id)+"/label", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveLabel removes a single label from a piece of content.
func (c *Client) RemoveLabel(ctx context.Context, id, name string) error {
	q := url.Values{}
	q.Set("name", name)
	return c.doJSON(ctx, "DELETE", "/rest/api/content/"+url.PathEscape(id)+"/label", q, nil, nil)
}
