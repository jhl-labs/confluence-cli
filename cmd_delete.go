package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
)

func runDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		id  = fs.String("id", "", "page ID to delete (required)")
		yes = fs.Bool("yes", false, "skip the confirmation prompt")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli delete --id PAGE_ID [--yes]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return err
	}

	cl, err := common.client()
	if err != nil {
		return err
	}
	ctx := context.Background()

	// Show what is about to be deleted and confirm, unless --yes.
	if !*yes {
		page, err := cl.Get(ctx, *id, nil)
		if err != nil {
			return err
		}
		title := page.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(os.Stderr, "About to delete page %s: %q\nType 'yes' to confirm: ", *id, title)
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(line) != "yes" {
			return fmt.Errorf("aborted")
		}
	}

	if err := cl.DeletePage(ctx, *id); err != nil {
		return err
	}

	result := map[string]string{"id": *id, "status": "deleted"}
	return emit(common.output, result, func() {
		fmt.Printf("deleted page %s\n", *id)
	})
}
