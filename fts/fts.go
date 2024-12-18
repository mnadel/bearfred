package fts

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mnadel/freddiebear/db"
	"github.com/pkg/errors"

	"database/sql"
)

const (
	dbFile = `fts.sqlite3`

	createSQL = `
		DROP TABLE IF EXISTS notes;

		CREATE VIRTUAL TABLE
		IF NOT EXISTS notes
		USING FTS5(title, body, uuid, tags, tokenize="trigram");
	`

	insertSQL = `
        INSERT INTO notes(title, body, uuid, tags)
        VALUES (?, ?, ?, ?)
	`

	selectSQL = `
		SELECT title, uuid, tags
        FROM notes
        WHERE (title MATCH '{param}') OR (body MATCH '{param}')
        ORDER BY RANK
	`

	readOnlyPragmaSQL = `
		PRAGMA query_only = on;
		PRAGMA synchronous = off;
		PRAGMA mmap_size = 250000000;
		PRAGMA temp_store = memory;
		PRAGMA journal_mode = off;
		PRAGMA cache_size = -25000;
	`

	infoSQL = `
		SELECT
			count(*) as record_count
		FROM
			notes
	`
)

type FTS struct {
	fts    *sql.DB
	bear   *db.DB
	dbFile string
}

func NewFTS(bear *db.DB) (*FTS, error) {
	var err error

	dbPath, found := os.LookupEnv("alfred_workflow_data'")
	if !found {
		dbPath, err = os.UserCacheDir()
		if err != nil {
			dbPath, err = os.UserHomeDir()
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}
	}

	fullDbPath := path.Join(dbPath, dbFile)

	db, err := sql.Open("sqlite3", fullDbPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &FTS{db, bear, fullDbPath}, nil
}

func (f *FTS) Close() {
	f.fts.Close()
}

func (f *FTS) Info() string {
	var err error
	bearRecordCount := 0
	count := -1

	row := f.fts.QueryRow(infoSQL)
	if row != nil {
		err = row.Scan(&count)
	}

	f.bear.Export(func(r *db.Record) error {
		bearRecordCount++
		return nil
	})

	return fmt.Sprintf(`    DB File: %s
FTS Records: %v
      Error: %v
 DB Records: %v`, f.dbFile, count, err, bearRecordCount)
}

func (f *FTS) Reindex() error {
	_, err := f.fts.Exec(createSQL)
	if err != nil {
		return err
	}

	var lastError error

	f.bear.Export(func(r *db.Record) error {
		_, err := f.fts.Exec(insertSQL, r.Title, r.Text, r.GUID, r.Tags)
		lastError = err
		return err
	})

	return lastError
}

func (f *FTS) Search(query string) (db.Results, error) {
	// see https://sqlite.org/fts5.html
	// regex magic:
	// 	\b -> matches a word transition boundary, from non-word char to a word char or vice-versa
	// 	\bXYZ\b -> XYZ begins and ends with a word boundary (i.e. is a word)
	//	(?i) -> ignore case
	//	near\( -> near isn't a standalone word, it's a function
	var re = regexp.MustCompile(`(\b(?i)near\(|\b(?i)and\b|\b(?i)or\b|\b(?i)not\b)`)
	normalized := re.ReplaceAllStringFunc(query, func(w string) string {
		return strings.ToUpper(w)
	})

	sql := strings.ReplaceAll(selectSQL, "{param}", normalized)

	records := make([]*db.Result, 0)

	_, err := f.fts.Exec(readOnlyPragmaSQL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rows, err := f.fts.Query(sql)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var guid, title, tags string

	for rows.Next() {
		err := rows.Scan(&title, &guid, &tags)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		record := &db.Result{
			NoteSHA: guid,
			ID:      guid,
			Title:   title,
			Tags:    tags,
		}

		records = append(records, record)
	}

	return records, nil
}
