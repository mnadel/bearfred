package fts

import (
	"fmt"
	"os"

	"github.com/mnadel/freddiebear/alfred"
	"github.com/mnadel/freddiebear/db"
	"github.com/mnadel/freddiebear/fts"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	optIndex bool
	optInfo  bool
)

func New() *cobra.Command {
	searchCmd := &cobra.Command{
		Use:   "fts [query]",
		Short: "Search for a note using SQLite3 FTS",
		Long:  "Search notes using full-text searching",
		RunE:  runner,
	}

	searchCmd.Flags().BoolVar(&optIndex, "index", false, "(re)index the database")
	searchCmd.Flags().BoolVar(&optInfo, "info", false, "show database info")

	return searchCmd
}

func runner(cmd *cobra.Command, args []string) error {
	if !optIndex && !optInfo && (len(args) != 1 || args[0] == "") {
		cmd.PrintErrln("missing arguments")
		os.Exit(1)
	}

	bearDB, err := db.NewDB()
	if err != nil {
		return errors.WithStack(err)
	}
	defer bearDB.Close()

	ftsDB, err := fts.NewFTS(bearDB)
	if err != nil {
		return errors.WithStack(err)
	}
	defer ftsDB.Close()

	if optInfo {
		fmt.Println(ftsDB.Info())
		return nil
	} else if optIndex {
		return ftsDB.Reindex()
	}

	results, err := ftsDB.Search(args[0])
	if err != nil {
		return errors.WithStack(err)
	}

	if len(results) == 0 {
		fmt.Print(alfred.AlfredCreateXML(args[0]))
	} else {
		fmt.Print(alfred.AlfredOpenXML(results, true))
	}

	return nil
}
