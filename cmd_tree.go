package main

import (
	"context"
	"flag"
	"fmt"

	"confluence-cli/internal/confluence"
)

// treeNode is the JSON-friendly nested form of a page subtree.
type treeNode struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Truncated bool        `json:"truncated,omitempty"` // children beyond the per-level limit were omitted
	Children  []*treeNode `json:"children,omitempty"`
}

func runTree(args []string) error {
	fs := flag.NewFlagSet("tree", flag.ContinueOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		space    = fs.String("space", common.defaultSpace, "space key to render from its root (default: CONFLUENCE_SPACE)")
		id       = fs.String("id", "", "page ID to render a subtree from (overrides --space)")
		depth    = fs.Int("depth", 5, "maximum tree depth to descend")
		perLevel = fs.Int("per-level", 100, "maximum children fetched per page")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli tree (--space KEY | --id PAGE_ID) [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == "" && *space == "" {
		return fmt.Errorf("provide --id or --space (or set CONFLUENCE_SPACE)")
	}

	cl, err := common.client()
	if err != nil {
		return err
	}
	ctx := context.Background()

	roots, err := treeRoots(ctx, cl, *id, *space, *perLevel)
	if err != nil {
		return err
	}

	nodes := make([]*treeNode, 0, len(roots))
	for _, r := range roots {
		n, err := buildSubtree(ctx, cl, r, *depth, *perLevel)
		if err != nil {
			return err
		}
		nodes = append(nodes, n)
	}

	return emit(common.output, nodes, func() {
		if len(nodes) == 0 {
			fmt.Println("(empty)")
			return
		}
		for _, n := range nodes {
			printTree(n, "", true, true)
		}
	})
}

// treeRoots resolves the starting page(s): an explicit page, a space homepage,
// or a space's top-level pages.
func treeRoots(ctx context.Context, cl *confluence.Client, id, space string, perLevel int) ([]confluence.Content, error) {
	if id != "" {
		c, err := cl.Get(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		return []confluence.Content{*c}, nil
	}

	// For a space, start from all top-level pages (depth=root) — this mirrors
	// the full page tree shown in the Confluence UI, including the homepage.
	rootPages, err := cl.GetSpaceRootPages(ctx, space, perLevel)
	if err != nil {
		return nil, err
	}
	return rootPages.Results, nil
}

func buildSubtree(ctx context.Context, cl *confluence.Client, page confluence.Content, depth, perLevel int) (*treeNode, error) {
	node := &treeNode{ID: page.ID, Title: page.Title}
	if depth <= 0 {
		return node, nil
	}
	children, err := cl.GetChildPages(ctx, page.ID, perLevel, 0)
	if err != nil {
		return nil, err
	}
	if children.Size > len(children.Results) {
		node.Truncated = true
	}
	for _, ch := range children.Results {
		sub, err := buildSubtree(ctx, cl, ch, depth-1, perLevel)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, sub)
	}
	return node, nil
}

// printTree renders a node with box-drawing connectors.
func printTree(n *treeNode, prefix string, isLast, isRoot bool) {
	if isRoot {
		fmt.Printf("%s  (%s)\n", n.Title, n.ID)
	} else {
		branch := "├── "
		if isLast {
			branch = "└── "
		}
		fmt.Printf("%s%s%s  (%s)\n", prefix, branch, n.Title, n.ID)
		if isLast {
			prefix += "    "
		} else {
			prefix += "│   "
		}
	}
	for i, ch := range n.Children {
		printTree(ch, prefix, i == len(n.Children)-1, false)
	}
	if n.Truncated {
		fmt.Printf("%s    … (more children omitted)\n", prefix)
	}
}
