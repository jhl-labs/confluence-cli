package main

import (
	"context"
	"flag"
	"fmt"
)

func runChildren(args []string) error {
	fs := flag.NewFlagSet("children", flag.ContinueOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		id    = fs.String("id", "", "parent page ID (required)")
		limit = fs.Int("limit", 100, "maximum children to return")
		start = fs.Int("start", 0, "pagination start offset")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli children --id PAGE_ID [flags]")
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

	res, err := cl.GetChildPages(context.Background(), *id, *limit, *start)
	if err != nil {
		return err
	}

	return emit(common.output, res, func() {
		if len(res.Results) == 0 {
			fmt.Println("no child pages")
			return
		}
		for _, c := range res.Results {
			fmt.Printf("%-12s %s\n", c.ID, c.Title)
		}
		fmt.Printf("\n%d child page(s)\n", len(res.Results))
	})
}
