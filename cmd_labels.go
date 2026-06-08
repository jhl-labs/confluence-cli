package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
)

func runLabels(args []string) error {
	fs := flag.NewFlagSet("labels", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		id     = fs.String("id", "", "page ID (required)")
		add    = fs.String("add", "", "comma-separated labels to add")
		remove = fs.String("remove", "", "comma-separated labels to remove")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli labels --id PAGE_ID [--add a,b] [--remove c]")
		fmt.Fprintln(fs.Output(), "  With no --add/--remove, lists current labels.")
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

	for _, name := range splitCSV(*remove) {
		if err := cl.RemoveLabel(ctx, *id, name); err != nil {
			return fmt.Errorf("removing label %q: %w", name, err)
		}
	}
	if names := splitCSV(*add); len(names) > 0 {
		if _, err := cl.AddLabels(ctx, *id, names); err != nil {
			return fmt.Errorf("adding labels: %w", err)
		}
	}

	// Always report the resulting label set.
	res, err := cl.GetLabels(ctx, *id)
	if err != nil {
		return err
	}
	return emit(common.output, res, func() {
		if len(res.Results) == 0 {
			fmt.Println("(no labels)")
			return
		}
		names := make([]string, 0, len(res.Results))
		for _, l := range res.Results {
			names = append(names, l.Name)
		}
		fmt.Println(strings.Join(names, ", "))
	})
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
