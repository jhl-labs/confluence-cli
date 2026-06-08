package main

import (
	"context"
	"flag"
	"fmt"

	"confluence-cli/internal/confluence"
)

func runCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		space    = fs.String("space", common.defaultSpace, "space key (required; default: CONFLUENCE_SPACE)")
		title    = fs.String("title", "", "page title (required)")
		bodyVal  = fs.String("body", "", "page body content")
		bodyFile = fs.String("body-file", "", `read body from file ("-" for stdin)`)
		rep      = fs.String("representation", "storage", "body format: storage|wiki")
		parent   = fs.String("parent", "", "parent page ID (optional)")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli create --space KEY --title T (--body B | --body-file F) [flags]")
		fmt.Fprintln(fs.Output(), "\nNote: Confluence bodies are XHTML 'storage' format, not Markdown.")
		fmt.Fprintln(fs.Output(), "Use --representation wiki to send Confluence wiki markup (converted server-side).")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("space", *space); err != nil {
		return err
	}
	if err := requireFlag("title", *title); err != nil {
		return err
	}

	bodyContent, err := readBody(*bodyVal, *bodyFile)
	if err != nil {
		return err
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	content, err := cl.Create(context.Background(), confluence.CreateInput{
		SpaceKey:       *space,
		Title:          *title,
		Body:           bodyContent,
		Representation: *rep,
		ParentID:       *parent,
	})
	if err != nil {
		return err
	}

	return emit(common.output, content, func() {
		fmt.Printf("created page %s\n", content.ID)
		if url := cl.WebURL(*content); url != "" {
			fmt.Println(url)
		}
	})
}
