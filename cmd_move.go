package main

import (
	"context"
	"flag"
	"fmt"
)

func runMove(args []string) error {
	fs := flag.NewFlagSet("move", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		id     = fs.String("id", "", "page ID to move (required)")
		parent = fs.String("parent", "", "new parent page ID (required)")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli move --id PAGE_ID --parent NEW_PARENT_ID")
		fmt.Fprintln(fs.Output(), `
Re-parent a page within the space hierarchy (Server/DC compatible):
  - Demote  : --parent <a sibling page>   (becomes a child of that sibling)
  - Promote : --parent <the grandparent>  (moves up one level)
The title and body are preserved; the version is auto-incremented.

Note: moving a page to the very top level (no parent) is not supported via the
REST API on Confluence Server/Data Center — use the Confluence UI for that.`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return err
	}
	if err := requireFlag("parent", *parent); err != nil {
		return err
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	moved, err := cl.MovePage(context.Background(), *id, *parent)
	if err != nil {
		return err
	}

	return emit(common.output, moved, func() {
		fmt.Printf("moved page %s under parent %s\n", *id, *parent)
		if url := cl.WebURL(*moved); url != "" {
			fmt.Println(url)
		}
	})
}
