package main

import (
	"context"
	"flag"
	"fmt"
)

func runGet(args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		id     = fs.String("id", "", "page ID (required)")
		expand = fs.String("expand", "space,version,body.storage", "comma-separated fields to expand")
		body   = fs.Bool("body", false, "print only the storage-format body (text output)")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli get --id PAGE_ID [flags]")
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

	content, err := cl.Get(context.Background(), *id, splitExpand(*expand))
	if err != nil {
		return err
	}

	return emit(common.output, content, func() {
		if *body {
			if content.Body != nil && content.Body.Storage != nil {
				fmt.Println(content.Body.Storage.Value)
			}
			return
		}
		printContentLine(cl, *content)
		if content.Version != nil {
			fmt.Printf("version:     %d\n", content.Version.Number)
		}
	})
}
