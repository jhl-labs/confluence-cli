package main

import (
	"context"
	"flag"
	"fmt"
)

func runSpaces(args []string) error {
	fs := flag.NewFlagSet("spaces", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		spaceType = fs.String("type", "", "filter by type: global|personal")
		limit     = fs.Int("limit", 50, "maximum spaces to return")
		start     = fs.Int("start", 0, "pagination start offset")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli spaces [--type global|personal] [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	res, err := cl.ListSpaces(context.Background(), *spaceType, *limit, *start)
	if err != nil {
		return err
	}

	return emit(common.output, res, func() {
		if len(res.Results) == 0 {
			fmt.Println("no spaces")
			return
		}
		for _, s := range res.Results {
			fmt.Printf("%-16s %-8s %s\n", s.Key, s.Type, s.Name)
		}
		fmt.Printf("\n%d space(s)\n", len(res.Results))
	})
}
