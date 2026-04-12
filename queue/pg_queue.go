package queue

import (
	"database/sql"
	"log/slog"

	"github.com/iamdanhart/te-live/catalog"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PgQueue struct {
	db *sql.DB
}

func NewPgQueue(dsn string) (*PgQueue, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PgQueue{db: db}, nil
}

func (q *PgQueue) Entries() []Entry {
	rows, err := q.db.Query(`
		SELECT qe.id, qe.name, s.id, s.title, s.artist, s.tab_url, es.performed
		FROM queue_entries qe
		JOIN entry_songs es ON es.entry_id = qe.id
		JOIN songs s ON s.id = es.song_id
		ORDER BY qe.position ASC, es.sort_order ASC`)
	if err != nil {
		slog.Error("Entries query", "err", err)
		return nil
	}
	defer rows.Close()
	return scanEntries(rows)
}

func (q *PgQueue) Current() *Entry {
	entries := q.Entries()
	if len(entries) == 0 {
		return nil
	}
	return &entries[0]
}

func (q *PgQueue) Next() *Entry {
	entries := q.Entries()
	if len(entries) < 2 {
		return nil
	}
	return &entries[1]
}

func (q *PgQueue) SignupsOpen() bool {
	var value string
	err := q.db.QueryRow(`SELECT value FROM settings WHERE key = 'signups_open'`).Scan(&value)
	if err != nil {
		slog.Error("SignupsOpen query", "err", err)
		return false
	}
	return value == "true"
}

func (q *PgQueue) ToggleSignups() bool {
	var value string
	err := q.db.QueryRow(`
		UPDATE settings
		SET value = CASE WHEN value = 'true' THEN 'false' ELSE 'true' END
		WHERE key = 'signups_open'
		RETURNING value`).Scan(&value)
	if err != nil {
		slog.Error("ToggleSignups query", "err", err)
		return false
	}
	return value == "true"
}

func (q *PgQueue) Add(name string, songs []catalog.Song) {
	tx, err := q.db.Begin()
	if err != nil {
		slog.Error("Add begin tx", "err", err)
		return
	}
	defer tx.Rollback()

	var entryID int
	err = tx.QueryRow(`
		INSERT INTO queue_entries (name, position)
		VALUES ($1, COALESCE((SELECT MAX(position) FROM queue_entries), 0) + 1)
		RETURNING id`, name).Scan(&entryID)
	if err != nil {
		slog.Error("Add insert entry", "err", err)
		return
	}

	for i, song := range songs {
		_, err = tx.Exec(`
			INSERT INTO entry_songs (entry_id, song_id, sort_order)
			VALUES ($1, (SELECT id FROM songs WHERE title = $2 AND artist = $3), $4)`,
			entryID, song.Title, song.Artist, i)
		if err != nil {
			slog.Error("Add insert entry_song", "err", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Add commit", "err", err)
	}
}

func (q *PgQueue) MoveCurrentToBottom() {
	_, err := q.db.Exec(`
		UPDATE queue_entries
		SET position = (SELECT MAX(position) FROM queue_entries) + 1
		WHERE id = (SELECT id FROM queue_entries ORDER BY position ASC LIMIT 1)`)
	if err != nil {
		slog.Error("MoveCurrentToBottom", "err", err)
	}
}

func (q *PgQueue) RemoveCurrent() {
	_, err := q.db.Exec(`
		DELETE FROM queue_entries
		WHERE id = (SELECT id FROM queue_entries ORDER BY position ASC LIMIT 1)`)
	if err != nil {
		slog.Error("RemoveCurrent", "err", err)
	}
}

func (q *PgQueue) MarkSongPerformed(title, artist string) {
	_, err := q.db.Exec(`
		UPDATE entry_songs SET performed = true
		WHERE entry_id = (SELECT id FROM queue_entries ORDER BY position ASC LIMIT 1)
		  AND song_id = (SELECT id FROM songs WHERE title = $1 AND artist = $2)`,
		title, artist)
	if err != nil {
		slog.Error("MarkSongPerformed", "err", err)
	}
}

func (q *PgQueue) RecordPerformed(singer string, song catalog.Song) {
	_, err := q.db.Exec(`
		INSERT INTO performed_songs (singer, song_id)
		VALUES ($1, (SELECT id FROM songs WHERE title = $2 AND artist = $3))`,
		singer, song.Title, song.Artist)
	if err != nil {
		slog.Error("RecordPerformed", "err", err)
	}
}

func (q *PgQueue) Performed() []PerformedSong {
	rows, err := q.db.Query(`
		SELECT ps.singer, s.id, s.title, s.artist, s.tab_url
		FROM performed_songs ps
		JOIN songs s ON s.id = ps.song_id
		ORDER BY ps.performed_at ASC`)
	if err != nil {
		slog.Error("Performed query", "err", err)
		return nil
	}
	defer rows.Close()

	var result []PerformedSong
	for rows.Next() {
		var ps PerformedSong
		if err := rows.Scan(&ps.Singer, &ps.Song.ID, &ps.Song.Title, &ps.Song.Artist, &ps.Song.TabUrl); err != nil {
			slog.Error("Performed scan", "err", err)
			continue
		}
		result = append(result, ps)
	}
	return result
}

func (q *PgQueue) AddSongToFirst(song catalog.Song) {
	_, err := q.db.Exec(`
		INSERT INTO entry_songs (entry_id, song_id, sort_order)
		VALUES (
			(SELECT id FROM queue_entries ORDER BY position ASC LIMIT 1),
			(SELECT id FROM songs WHERE title = $1 AND artist = $2),
			COALESCE((SELECT MAX(sort_order) FROM entry_songs
			          WHERE entry_id = (SELECT id FROM queue_entries ORDER BY position ASC LIMIT 1)), 0) + 1
		)`, song.Title, song.Artist)
	if err != nil {
		slog.Error("AddSongToFirst", "err", err)
	}
}

// scanEntries collapses the joined rows into []Entry, grouping songs by singer.
func scanEntries(rows *sql.Rows) []Entry {
	var entries []Entry
	entryIndex := map[int]int{} // queue_entry id → index in entries slice

	for rows.Next() {
		var (
			entryID   int
			name      string
			songID    int
			title     string
			artist    string
			tabUrl    string
			performed bool
		)
		if err := rows.Scan(&entryID, &name, &songID, &title, &artist, &tabUrl, &performed); err != nil {
			slog.Error("scanEntries scan", "err", err)
			continue
		}
		idx, exists := entryIndex[entryID]
		if !exists {
			entries = append(entries, Entry{Name: name})
			idx = len(entries) - 1
			entryIndex[entryID] = idx
		}
		entries[idx].Songs = append(entries[idx].Songs, SongEntry{
			Song:      catalog.Song{ID: songID, Title: title, Artist: artist, TabUrl: tabUrl},
			Performed: performed,
		})
	}
	return entries
}