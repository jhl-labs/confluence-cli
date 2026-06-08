package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
)

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		text   = fs.String("text", "", "free-text search (matched against title and body)")
		cql    = fs.String("cql", "", "raw CQL query (overrides --text and --space)")
		space  = fs.String("space", common.defaultSpace, "restrict to a space key (default: CONFLUENCE_SPACE)")
		limit  = fs.Int("limit", 25, "maximum results")
		expand = fs.String("expand", "space,version", "comma-separated fields to expand")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli search [--text QUERY | --cql QUERY] [--space KEY] [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := *cql
	if query == "" {
		query = buildCQL(*text, *space)
	}
	if query == "" {
		return fmt.Errorf("provide --text, --cql, or --space")
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	res, err := cl.Search(context.Background(), query, *limit, splitExpand(*expand))
	if err != nil {
		return err
	}

	return emit(common.output, res, func() {
		if len(res.Results) == 0 {
			fmt.Println("no results")
			return
		}
		for _, c := range res.Results {
			printContentLine(cl, c)
		}
		fmt.Printf("\n%d result(s)\n", len(res.Results))
	})
}

// buildCQL composes a simple CQL query from a text term and/or space key.
func buildCQL(text, space string) string {
	var clauses []string
	if space != "" {
		clauses = append(clauses, fmt.Sprintf("space = %s", cqlQuote(space)))
	}
	if text != "" {
		clauses = append(clauses, fmt.Sprintf("text ~ %s", cqlQuote(text)))
	}
	return strings.Join(clauses, " AND ")
}

// cqlQuote wraps a value in double quotes, escaping embedded quotes/backslashes
// so it is safe to interpolate into a CQL expression.
func cqlQuote(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`
}

func splitExpand(s string) []string {
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
