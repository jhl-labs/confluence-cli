package confluence

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Representation is the body format for create/update operations.
//   - "storage" : Confluence XHTML storage format (the canonical format)
//   - "wiki"    : Confluence wiki markup, converted server-side on write
const (
	RepresentationStorage = "storage"
	RepresentationWiki    = "wiki"
)

// Content is a Confluence content object (page, blogpost, or comment).
type Content struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`
	Status  string            `json:"status,omitempty"`
	Title   string            `json:"title,omitempty"`
	Space   *Space            `json:"space,omitempty"`
	Body    *Body             `json:"body,omitempty"`
	Version *Version          `json:"version,omitempty"`
	Links   map[string]string `json:"_links,omitempty"`
}

// Space is a Confluence space reference.
type Space struct {
	Key      string   `json:"key,omitempty"`
	Name     string   `json:"name,omitempty"`
	Type     string   `json:"type,omitempty"`
	Homepage *Content `json:"homepage,omitempty"`
}

// Version tracks the content revision number.
type Version struct {
	Number int `json:"number"`
}

// Body holds one or more rendered representations of content.
type Body struct {
	Storage *BodyValue `json:"storage,omitempty"`
	View    *BodyValue `json:"view,omitempty"`
}

// BodyValue is a single body representation.
type BodyValue struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// SearchResult is the response from a content search.
type SearchResult struct {
	Results []Content `json:"results"`
	Size    int       `json:"size"`
	Limit   int       `json:"limit"`
	Start   int       `json:"start"`
}

// WebURL returns the absolute browser URL for the content, if known.
func (c *Client) WebURL(content Content) string {
	if content.Links == nil {
		return ""
	}
	webui := content.Links["webui"]
	if webui == "" {
		return ""
	}
	// _links.webui is relative to the context path; base also carries it.
	return c.baseURL + webui
}

// Search runs a CQL query against /rest/api/content/search.
func (c *Client) Search(ctx context.Context, cql string, limit int, expand []string) (*SearchResult, error) {
	q := url.Values{}
	q.Set("cql", cql)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if len(expand) > 0 {
		q.Set("expand", strings.Join(expand, ","))
	}
	var out SearchResult
	if err := c.doJSON(ctx, "GET", "/rest/api/content/search", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches a single content object by ID.
func (c *Client) Get(ctx context.Context, id string, expand []string) (*Content, error) {
	q := url.Values{}
	if len(expand) > 0 {
		q.Set("expand", strings.Join(expand, ","))
	}
	var out Content
	if err := c.doJSON(ctx, "GET", "/rest/api/content/"+url.PathEscape(id), q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateInput describes a new page.
type CreateInput struct {
	SpaceKey       string
	Title          string
	Body           string
	Representation string // storage (default) or wiki
	ParentID       string // optional ancestor
}

// Create makes a new page.
func (c *Client) Create(ctx context.Context, in CreateInput) (*Content, error) {
	rep := in.Representation
	if rep == "" {
		rep = RepresentationStorage
	}

	payload := map[string]any{
		"type":  "page",
		"title": in.Title,
		"space": map[string]string{"key": in.SpaceKey},
		"body": map[string]any{
			rep: map[string]string{
				"value":          in.Body,
				"representation": rep,
			},
		},
	}
	if in.ParentID != "" {
		payload["ancestors"] = []map[string]string{{"id": in.ParentID}}
	}

	var out Content
	if err := c.doJSON(ctx, "POST", "/rest/api/content", nil, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateInput describes a page edit. The next version number is resolved
// automatically from the current content unless Version is set explicitly.
type UpdateInput struct {
	ID             string
	Title          string // optional; keeps current title if empty
	Body           string
	Representation string // storage (default) or wiki
	Version        int    // optional; 0 means "current + 1"
}

// Update edits an existing page, incrementing its version.
func (c *Client) Update(ctx context.Context, in UpdateInput) (*Content, error) {
	rep := in.Representation
	if rep == "" {
		rep = RepresentationStorage
	}

	// Resolve current title/version when not fully specified.
	current, err := c.Get(ctx, in.ID, []string{"version"})
	if err != nil {
		return nil, fmt.Errorf("fetching current page: %w", err)
	}
	nextVersion := in.Version
	if nextVersion == 0 {
		if current.Version == nil {
			return nil, fmt.Errorf("could not determine current version of %s", in.ID)
		}
		nextVersion = current.Version.Number + 1
	}
	title := in.Title
	if title == "" {
		title = current.Title
	}

	payload := map[string]any{
		"id":      in.ID,
		"type":    current.Type,
		"title":   title,
		"version": map[string]int{"number": nextVersion},
		"body": map[string]any{
			rep: map[string]string{
				"value":          in.Body,
				"representation": rep,
			},
		},
	}

	var out Content
	if err := c.doJSON(ctx, "PUT", "/rest/api/content/"+url.PathEscape(in.ID), nil, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddComment posts a comment on a page.
func (c *Client) AddComment(ctx context.Context, pageID, body, representation string) (*Content, error) {
	rep := representation
	if rep == "" {
		rep = RepresentationStorage
	}
	payload := map[string]any{
		"type":      "comment",
		"container": map[string]string{"id": pageID, "type": "page"},
		"body": map[string]any{
			rep: map[string]string{
				"value":          body,
				"representation": rep,
			},
		},
	}
	var out Content
	if err := c.doJSON(ctx, "POST", "/rest/api/content", nil, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
