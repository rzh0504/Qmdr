package history

import (
	"context"
	"time"

	"github.com/egoist/mygo/plugins/sqlite"
)

type QueueRecord struct {
	ID          int
	SongMID     string
	Title       string
	Artist      string
	Album       string
	AlbumMID    string
	Duration    string
	SongQuality string
	VIP         bool
	Status      int
	Progress    float64
	Quality     string
	Error       string
	FilePath    string
	FileSize    int64
	CreatedAt   time.Time
}

const queueUpsertSQL = `
	INSERT INTO queue_items
		(id, song_mid, title, artist, album, album_mid, duration, song_quality, vip, status,
		 progress, quality, error, file_path, file_size, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		song_mid = excluded.song_mid,
		title = excluded.title,
		artist = excluded.artist,
		album = excluded.album,
		album_mid = excluded.album_mid,
		duration = excluded.duration,
		song_quality = excluded.song_quality,
		vip = excluded.vip,
		status = excluded.status,
		progress = excluded.progress,
		quality = excluded.quality,
		error = excluded.error,
		file_path = excluded.file_path,
		file_size = excluded.file_size,
		created_at = excluded.created_at`

func queueArgs(record QueueRecord) []any {
	createdAt := record.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	vip := 0
	if record.VIP {
		vip = 1
	}
	return []any{
		record.ID,
		record.SongMID,
		record.Title,
		record.Artist,
		record.Album,
		record.AlbumMID,
		record.Duration,
		record.SongQuality,
		vip,
		record.Status,
		record.Progress,
		record.Quality,
		record.Error,
		record.FilePath,
		record.FileSize,
		createdAt.Unix(),
	}
}

func (s *Store) SaveQueueItems(ctx context.Context, records []QueueRecord) error {
	if len(records) == 0 {
		return nil
	}
	statements := make([]sqlite.Statement, 0, len(records))
	for _, record := range records {
		statements = append(statements, sqlite.Statement{SQL: queueUpsertSQL, Args: queueArgs(record)})
	}
	return s.withDB(func(db *sqlite.DB) error {
		_, err := db.Transaction(ctx, statements)
		return err
	})
}

func (s *Store) SaveQueueItem(ctx context.Context, record QueueRecord) error {
	return s.SaveQueueItems(ctx, []QueueRecord{record})
}

func (s *Store) ReplaceQueue(ctx context.Context, records []QueueRecord) error {
	statements := make([]sqlite.Statement, 0, len(records)+1)
	statements = append(statements, sqlite.Statement{SQL: `DELETE FROM queue_items`})
	for _, record := range records {
		statements = append(statements, sqlite.Statement{SQL: queueUpsertSQL, Args: queueArgs(record)})
	}
	return s.withDB(func(db *sqlite.DB) error {
		_, err := db.Transaction(ctx, statements)
		return err
	})
}

func (s *Store) DeleteQueueItems(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	statements := make([]sqlite.Statement, 0, len(ids))
	for _, id := range ids {
		statements = append(statements, sqlite.Statement{
			SQL:  `DELETE FROM queue_items WHERE id = ?`,
			Args: []any{id},
		})
	}
	return s.withDB(func(db *sqlite.DB) error {
		_, err := db.Transaction(ctx, statements)
		return err
	})
}

func (s *Store) PruneQueue(ctx context.Context, before time.Time) error {
	return s.withDB(func(db *sqlite.DB) error {
		_, err := db.Execute(ctx, `DELETE FROM queue_items WHERE created_at < ?`, before.Unix())
		return err
	})
}

func (s *Store) LoadQueue(ctx context.Context, since time.Time) ([]QueueRecord, error) {
	var records []QueueRecord
	err := s.withDB(func(db *sqlite.DB) error {
		rows, err := db.Query(ctx, `
			SELECT id, song_mid, title, artist, album, album_mid, duration, song_quality, vip, status,
			       progress, quality, error, file_path, file_size, created_at
			FROM queue_items
			WHERE created_at >= ?
			ORDER BY id ASC`, since.Unix())
		if err != nil {
			return err
		}
		records = make([]QueueRecord, 0, len(rows.Values))
		for _, row := range rows.Values {
			if len(row) < 16 {
				continue
			}
			records = append(records, QueueRecord{
				ID:          int(int64Value(row[0])),
				SongMID:     stringValue(row[1]),
				Title:       stringValue(row[2]),
				Artist:      stringValue(row[3]),
				Album:       stringValue(row[4]),
				AlbumMID:    stringValue(row[5]),
				Duration:    stringValue(row[6]),
				SongQuality: stringValue(row[7]),
				VIP:         int64Value(row[8]) != 0,
				Status:      int(int64Value(row[9])),
				Progress:    float64Value(row[10]),
				Quality:     stringValue(row[11]),
				Error:       stringValue(row[12]),
				FilePath:    stringValue(row[13]),
				FileSize:    int64Value(row[14]),
				CreatedAt:   time.Unix(int64Value(row[15]), 0).Local(),
			})
		}
		return nil
	})
	return records, err
}

func float64Value(value any) float64 {
	switch number := value.(type) {
	case float64:
		return number
	case int64:
		return float64(number)
	default:
		return 0
	}
}
