package db

import (
	"crypto/md5"
	"fmt"
	"os"
	"path"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mnadel/freddiebear/util"
	"github.com/pkg/errors"

	"database/sql"
)

const (
	dbFile = `/Library/Group Containers/9K33E3U3T4.net.shinyfrog.bear/Application Data/database.sqlite?mode=ro`

	sqlDeletedAttachments = `
		SELECT
			f.ZUNIQUEIDENTIFIER,
			f.ZFILENAME
		FROM
			ZSFNOTEFILE f 
			LEFT OUTER JOIN ZSFNOTE n on n.Z_PK = f.ZNOTE
		WHERE
			n.ZTRASHED = 1
			OR n.ZTRASHEDDATE IS NOT NULL
	`

	sqlNotesByTag = `
		SELECT
			note.ZUNIQUEIDENTIFIER,
			note.ZTITLE,
			datetime(note.ZMODIFICATIONDATE, 'unixepoch', '31 years', 'localtime') as mod_date,
			note.ZTEXT
		FROM
			ZSFNOTE note
			LEFT OUTER JOIN Z_5TAGS tags ON note.Z_PK = tags.Z_5NOTES
			LEFT OUTER JOIN ZSFNOTETAG tag ON tags.Z_13TAGS = tag.Z_PK
		WHERE
			note.ZARCHIVED = 0
			AND note.ZTRASHED = 0
			AND tag.ZTITLE = ?
		ORDER BY
			note.ZTITLE DESC
	`

	sqlTags = `
		SELECT DISTINCT
			note.Z_PK,
			GROUP_CONCAT(COALESCE(tag.ZTITLE, ''))
		FROM
			ZSFNOTE note
			LEFT OUTER JOIN Z_5TAGS tags ON note.Z_PK = tags.Z_5NOTES
			LEFT OUTER JOIN ZSFNOTETAG tag ON tags.Z_13TAGS = tag.Z_PK
		WHERE
			note.ZARCHIVED = 0
			AND note.ZTRASHED = 0
		GROUP BY
			note.Z_PK
	`

	sqlAllTags = `
		SELECT DISTINCT tag.ZTITLE
		FROM
			ZSFNOTE note
			LEFT OUTER JOIN Z_5TAGS tags ON note.Z_PK = tags.Z_5NOTES
			LEFT OUTER JOIN ZSFNOTETAG tag ON tags.Z_13TAGS = tag.Z_PK
		WHERE
			note.ZARCHIVED = 0
			AND note.ZTRASHED = 0
			AND tag.ZTITLE IS NOT NULL
	`

	sqlAllTitles = `
		SELECT DISTINCT
			note.ZUNIQUEIDENTIFIER,
			note.ZTITLE,
			GROUP_CONCAT(COALESCE(tag.ZTITLE, ''))
		FROM
			ZSFNOTE note
			LEFT OUTER JOIN Z_5TAGS tags ON note.Z_PK = tags.Z_5NOTES
			LEFT OUTER JOIN ZSFNOTETAG tag ON tags.Z_13TAGS = tag.Z_PK
		WHERE
			note.ZARCHIVED = 0
			AND note.ZTRASHED = 0
		GROUP BY
			note.ZUNIQUEIDENTIFIER
		ORDER BY
			note.ZMODIFICATIONDATE DESC
	`

	sqlTitle = `
		SELECT DISTINCT
			note.ZUNIQUEIDENTIFIER,
			note.ZTITLE,
			GROUP_CONCAT(COALESCE(tag.ZTITLE, ''))
		FROM
			ZSFNOTE note
			LEFT OUTER JOIN Z_5TAGS tags ON note.Z_PK = tags.Z_5NOTES
			LEFT OUTER JOIN ZSFNOTETAG tag ON tags.Z_13TAGS = tag.Z_PK
		WHERE
			note.ZARCHIVED = 0
			AND note.ZTRASHED = 0
			AND LOWER(note.ZTITLE) LIKE LOWER(?)
		GROUP BY
			note.ZUNIQUEIDENTIFIER
		ORDER BY
			note.ZMODIFICATIONDATE DESC
	`

	sqlText = `
		SELECT
			note.ZUNIQUEIDENTIFIER,
			note.ZTITLE,
			GROUP_CONCAT(COALESCE(tag.ZTITLE, ''))
		FROM
			ZSFNOTE note
			LEFT OUTER JOIN Z_5TAGS tags ON note.Z_PK = tags.Z_5NOTES
			LEFT OUTER JOIN ZSFNOTETAG tag ON tags.Z_13TAGS = tag.Z_PK
		WHERE
			note.ZARCHIVED = 0
			AND note.ZTRASHED = 0
			AND (LOWER(note.ZTEXT) LIKE LOWER(?) OR LOWER(note.ZTITLE) LIKE LOWER(?))
		GROUP BY
			note.ZUNIQUEIDENTIFIER
		ORDER BY
			note.ZMODIFICATIONDATE DESC
	`

	sqlExport = `
		SELECT
			note.ZUNIQUEIDENTIFIER,
			note.ZTITLE,
			note.ZTEXT,
			GROUP_CONCAT(COALESCE(tag.ZTITLE, ''))
		FROM
			ZSFNOTE note
			LEFT OUTER JOIN Z_5TAGS tags ON note.Z_PK = tags.Z_5NOTES
			LEFT OUTER JOIN ZSFNOTETAG tag ON tags.Z_13TAGS = tag.Z_PK
		WHERE
			ZARCHIVED = 0 
			and ZTRASHED = 0
		GROUP BY
			note.ZUNIQUEIDENTIFIER
	`

	sqlGraph = `
		SELECT
			DISTINCT
			src.Z_PK as sid,
			src.ZUNIQUEIDENTIFIER as suuid,
			src.ZTITLE as stitle,
			target.Z_PK as tid,
			target.ZUNIQUEIDENTIFIER as tuuid,
			target.ZTITLE as ttitle
		FROM
			ZSFNOTEBACKLINK b
			JOIN ZSFNOTE src ON src.Z_PK = b.ZLINKINGTO
			JOIN ZSFNOTE target ON target.Z_PK = b.ZLINKEDBY
		WHERE
			src.ZARCHIVED = 0
			AND src.ZTRASHED = 0
			AND target.ZARCHIVED = 0
			AND target.ZTRASHED = 0
	`

	sqlAttachments = `
		SELECT
			n.ZUNIQUEIDENTIFIER as note_uuid,
			n.ZTITLE as note_title,
			f.ZUNIQUEIDENTIFIER as folder_uuid,
			f.ZFILENAME as filename
		FROM
			ZSFNOTE n 
			JOIN ZSFNOTEFILE f on f.ZNOTE = n.Z_PK
		WHERE
			n.ZARCHIVED = 0
			AND n.ZTRASHED = 0
		ORDER BY
			n.ZUNIQUEIDENTIFIER
	`

	sqlPragma = `
		PRAGMA query_only = on;
		PRAGMA synchronous = off;
		PRAGMA mmap_size = 250000000;
		PRAGMA temp_store = memory;
		PRAGMA journal_mode = off;
		PRAGMA cache_size = -25000;
	`
)

// Exporter is a func that receives an exported record
type Exporter func(record *Record) error

// DB represents the Bear Notes database
type DB struct {
	db *sql.DB
}

// Record represents an exported note
type Record struct {
	SHA              string
	Title            string
	Text             string
	ModificationDate string
	GUID             string
	Tags             string
}

// Result references a specific note: its identifier and title
type Result struct {
	NoteSHA string
	ID      string
	Title   string
	Tags    string
}

type Attachment struct {
	NoteSHA    string
	NoteTitle  string
	FolderUUID string
	Filename   string
}

// Results is a list of *Result, and represents a collection of notes in the database
type Results []*Result

type Edge struct {
	Source *Result
	Target *Result
}

type Graph []*Edge

// Create a new DB, referencing the user's Bear Notes database
func NewDB() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	db, err := sql.Open("sqlite3", path.Join(home, dbFile))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := db.Exec(sqlPragma); err != nil {
		return nil, errors.WithStack(err)
	}

	return &DB{db}, nil
}

// Close cleans up our database connection
func (d *DB) Close() error {
	return d.db.Close()
}

// AllAttachments returns a list of all attachments in the database.
func (d *DB) AllAttachments() ([]*Attachment, error) {
	records := make([]*Attachment, 0)

	rows, err := d.db.Query(sqlAttachments)
	if err != nil {
		return nil, errors.WithStack(rows.Err())
	}

	var noteID, noteTitle, folderID, fileName string

	for rows.Next() {
		err := rows.Scan(&noteID, &noteTitle, &folderID, &fileName)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		records = append(records, &Attachment{
			NoteSHA:    guidToSHA(noteID),
			NoteTitle:  noteTitle,
			FolderUUID: folderID,
			Filename:   fileName,
		})
	}

	return records, nil
}

// Records returns the list of notes in the database
func (d *DB) Records() ([]*Record, error) {
	records := make([]*Record, 0)

	rows, err := d.db.Query(sqlExport)
	if err != nil {
		return nil, errors.WithStack(rows.Err())
	}

	var guid, title, text, tags string

	for rows.Next() {
		err := rows.Scan(&guid, &title, &text, &tags)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		record := &Record{
			SHA:   guidToSHA(guid),
			Title: title,
			Text:  text,
			GUID:  guid,
			Tags:  tags,
		}

		records = append(records, record)
	}

	return records, nil
}

// Export notes to specified directory
func (d *DB) Export(exporter Exporter) error {
	records, err := d.Records()
	if err != nil {
		return errors.WithStack(err)
	}

	for _, record := range records {
		if err = exporter(record); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// QueryTitles searches for a term within the titles of notes within the database, setting
// `exact` to true will do an exact match, else it'll perform a substring match
func (d *DB) QueryTitles(term string, exact bool) (Results, error) {
	var bind string

	if exact {
		bind = term
	} else {
		bind = substringSearch(term)
	}

	rows, err := d.db.Query(sqlTitle, bind)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	return RowsToResults(rows)
}

// QueryAllTitles returns a list of all titles
func (d *DB) QueryAllTitles() (Results, error) {
	rows, err := d.db.Query(sqlAllTitles)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	return RowsToResults(rows)
}

// QueryText searches for a term within the body or title of notes within the database.
func (d *DB) QueryText(term string) (Results, error) {
	bind := substringSearch(term)
	rows, err := d.db.Query(sqlText, bind, bind)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer rows.Close()

	return RowsToResults(rows)
}

// QueryTags returns a list of all tags
func (d *DB) QueryTags() ([]string, error) {
	rows, err := d.db.Query(sqlAllTags)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer rows.Close()

	tags := make([]string, 0)
	var tag string

	for rows.Next() {
		err := rows.Scan(&tag)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		tags = append(tags, tag)
	}

	return util.RemoveIntermediatePrefixes(tags, "/"), nil
}

// QueryDeletedAttachments returns a list of attachments that can be deleted off disk.
func (d *DB) QueryDeletedAttachments() ([]*Attachment, error) {
	rows, err := d.db.Query(sqlDeletedAttachments)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	attachments := make([]*Attachment, 0)
	var guid, filename string

	for rows.Next() {
		err := rows.Scan(&guid, &filename)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		attachments = append(attachments, &Attachment{
			FolderUUID: guid,
			Filename:   filename,
		})
	}

	return attachments, nil
}

// QueryTag searches for all notes with a given tag within the database.
func (d *DB) QueryTag(tag string) ([]*Record, error) {
	rows, err := d.db.Query(sqlNotesByTag, tag)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	records := make([]*Record, 0)
	var guid, title, text, moddate string

	for rows.Next() {
		err := rows.Scan(&guid, &title, &moddate, &text)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		records = append(records, &Record{guid, title, text, moddate, guid, ""})
	}

	return records, nil
}

// QueryGraph returns a graph of linked notes
func (d *DB) QueryGraph() (Graph, error) {
	tags, err := d.tagsByNoteID()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rows, err := d.db.Query(sqlGraph)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	var sourceID int
	var sourceUUID string
	var sourceTitle string
	var targetID int
	var targetUUID string
	var targetTitle string

	results := make(Graph, 0)

	for rows.Next() {
		err := rows.Scan(&sourceID, &sourceUUID, &sourceTitle, &targetID, &targetUUID, &targetTitle)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		results = append(results, &Edge{
			Source: &Result{
				ID:    sourceUUID,
				Title: sourceTitle,
				Tags:  tags[sourceID],
			},
			Target: &Result{
				ID:    targetUUID,
				Title: targetTitle,
				Tags:  tags[targetID],
			},
		})
	}

	return results, errors.WithStack(rows.Err())
}

func (d *DB) tagsByNoteID() (map[int]string, error) {
	tags := make(map[int]string)

	rows, err := d.db.Query(sqlTags)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	var noteID int
	var noteTags string

	for rows.Next() {
		err := rows.Scan(&noteID, &noteTags)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		tags[noteID] = noteTags
	}

	return tags, nil
}

// UniqueTags returns the leaf-node tags ([a a/b a/b/c d] -> [a/b/c d])
func (r *Result) UniqueTags() []string {
	split := strings.Split(r.Tags, ",")
	return util.RemoveIntermediatePrefixes(split, "/")
}

// TitleCase returns a Alfred-safe version of the proper title casing
func (r *Result) TitleCase() string {
	return util.ToSafeString(util.ToTitleCase(r.Title))
}

// RowsToResults will pakcage a SQL row of (id, title, tags)
func RowsToResults(rows *sql.Rows) (Results, error) {
	var id string
	var title string
	var tags string

	results := make(Results, 0)

	for rows.Next() {
		err := rows.Scan(&id, &title, &tags)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		results = append(results, &Result{
			NoteSHA: guidToSHA(id),
			ID:      id,
			Title:   title,
			Tags:    tags,
		})
	}

	return results, errors.WithStack(rows.Err())
}

func substringSearch(term string) string {
	bind := strings.Builder{}
	bind.WriteString(`%`)
	bind.WriteString(term)
	bind.WriteString(`%`)
	return bind.String()
}

func guidToSHA(guid string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(guid)))[0:7]
}
