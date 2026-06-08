package main

import (
	"context"
	"flag"
	"fmt"
)

func runComment(args []string) error {
	fs := flag.NewFlagSet("comment", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		pageID   = fs.String("id", "", "page ID to comment on (required)")
		bodyVal  = fs.String("body", "", "comment body content")
		bodyFile = fs.String("body-file", "", `read body from file ("-" for stdin)`)
		rep      = fs.String("representation", "storage", "body format: storage|wiki")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli comment --id PAGE_ID (--body B | --body-file F) [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("id", *pageID); err != nil {
		return err
	}

	bodyContent, err := readBody(*bodyVal, *bodyFile)
	if err != nil {
		return err
	}
	if bodyContent == "" {
		return fmt.Errorf("comment body is empty (use --body or --body-file)")
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	content, err := cl.AddComment(context.Background(), *pageID, bodyContent, *rep)
	if err != nil {
		return err
	}

	return emit(common.output, content, func() {
		fmt.Printf("added comment %s on page %s\n", content.ID, *pageID)
	})
}
