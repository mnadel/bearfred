package backlinks

import (
	"fmt"
	"strings"

	"github.com/mnadel/freddiebear/db"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type Source = *db.Result
type Target = *db.Result

func New() *cobra.Command {
	searchCmd := &cobra.Command{
		Use:   "backlinks [term]",
		Short: "Show backlinks for notes matching search term",
		Long:  "Generate backlink results in Alfred Workflow's XML schema format",
		Args:  cobra.ExactArgs(1),
		RunE:  runner,
	}

	return searchCmd
}

func runner(cmd *cobra.Command, args []string) error {
	bearDB, err := db.NewDB()
	if err != nil {
		return errors.WithStack(err)
	}
	defer bearDB.Close()

	graph, err := bearDB.QueryGraph()
	if err != nil {
		return errors.WithStack(err)
	}

	term := strings.ToLower(args[0])
	matches := make(map[Target]Source)

	for _, edge := range graph {
		if strings.Contains(strings.ToLower(edge.Target.Title), term) {
			matches[edge.Target] = edge.Source
		}
	}

	fmt.Println(buildOpenXml(matches))

	return nil
}

func buildOpenXml(matches map[Target]Source) string {
	builder := strings.Builder{}

	builder.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	builder.WriteString(`<items>`)

	if len(matches) == 0 {
		builder.WriteString(`<item valid="no"><title>No backlinks found</title></item>`)
	} else {
		for target, source := range matches {
			source.Title = fmt.Sprintf("%s → %s", source.Title, target.Title)

			builder.WriteString(`<item valid="yes">`)
			builder.WriteString(`<title>`)
			builder.WriteString(source.TitleCase())
			builder.WriteString(`</title>`)

			builder.WriteString(`<subtitle>`)
			builder.WriteString(strings.Join(source.UniqueTags(), ", "))
			builder.WriteString(`</subtitle>`)

			builder.WriteString(`<arg>`)
			builder.WriteString(source.ID)
			builder.WriteString(`</arg>`)
			builder.WriteString(`</item>`)
		}
	}

	builder.WriteString(`</items>`)

	return builder.String()
}
