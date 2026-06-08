package main

import (
	"context"
	"flag"
	"fmt"

	"confluence-cli/internal/confluence"
)

func runUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		id       = fs.String("id", "", "page ID (required)")
		title    = fs.String("title", "", "new title (optional; keeps current if empty)")
		bodyVal  = fs.String("body", "", "new body content")
		bodyFile = fs.String("body-file", "", `read body from file ("-" for stdin)`)
		rep      = fs.String("representation", "storage", "body format: storage|wiki")
		version  = fs.Int("version", 0, "explicit new version number (0 = current+1)")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli update --id PAGE_ID (--body B | --body-file F) [flags]")
		fmt.Fprintln(fs.Output(), "\nThe new version number is resolved automatically (current + 1) unless --version is set.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
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

	content, err := cl.Update(context.Background(), confluence.UpdateInput{
		ID:             *id,
		Title:          *title,
		Body:           bodyContent,
		Representation: *rep,
		Version:        *version,
	})
	if err != nil {
		return err
	}

	return emit(common.output, content, func() {
		v := 0
		if content.Version != nil {
			v = content.Version.Number
		}
		fmt.Printf("updated page %s to version %d\n", content.ID, v)
		if url := cl.WebURL(*content); url != "" {
			fmt.Println(url)
		}
	})
}
