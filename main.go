package main

import (
	"log"

	"github.com/mnadel/freddiebear/cmd/backlinks"
	"github.com/mnadel/freddiebear/cmd/cleanup"
	"github.com/mnadel/freddiebear/cmd/export"
	"github.com/mnadel/freddiebear/cmd/forwardlinks"
	"github.com/mnadel/freddiebear/cmd/fts"
	"github.com/mnadel/freddiebear/cmd/graph"
	"github.com/mnadel/freddiebear/cmd/journal"
	"github.com/mnadel/freddiebear/cmd/search"
	"github.com/mnadel/freddiebear/cmd/tags"
	"github.com/mnadel/freddiebear/cmd/titles"
	"github.com/mnadel/freddiebear/cmd/transcript"
	"github.com/mnadel/freddiebear/cmd/version"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:   "freddiebear",
		Short: "A CLI for an Alfred+Bear integration",
		Long:  "Search notes, plus helpers to implement a daily journal",
	}

	cmd.AddCommand(journal.New())
	cmd.AddCommand(search.New())
	cmd.AddCommand(version.New())
	cmd.AddCommand(export.New())
	cmd.AddCommand(graph.New())
	cmd.AddCommand(backlinks.New())
	cmd.AddCommand(forwardlinks.New())
	cmd.AddCommand(transcript.New())
	cmd.AddCommand(tags.New())
	cmd.AddCommand(cleanup.New())
	cmd.AddCommand(titles.New())
	cmd.AddCommand(fts.New())

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
