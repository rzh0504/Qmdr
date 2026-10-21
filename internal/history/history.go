package history

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/plugins/sqlite"
)

const (
	StatusDone    = "done"
	StatusSkipped = "skipped"
)

type Record struct {
	ID          int64
	SongMID     string
	Title       string
	Artist      string
	Album       string
	AlbumMID    string
	Quality     string
	FilePath    string
	FileSize    int64
	Status      string
	CompletedAt time.Time
}

type Stats struct {
	Count int
	Bytes int64
}

type Store struct {
	db *sqlite.DB
}

func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "qmdr.db")
}

func Open(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("找不到配置目录")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sqlite.Open(ctx, path, sqlite.OpenOptions{})
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.Execute(ctx, `
		CREATE TABLE IF NOT EXISTS downloads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			song_mid TEXT NOT NULL,
			title TEXT NOT NULL,
			artist TEXT NOT NULL,
			album TEXT NOT NULL,
			album_mid TEXT NOT NULL,
			quality TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			status TEXT NOT NULL,
			completed_at INTEGER NOT NULL
		)`); err != nil {
		return err
	}
	if _, err := s.db.Execute(ctx,
		`CREATE INDEX IF NOT EXISTS idx_downloads_completed_at ON downloads(completed_at DESC, id DESC)`); err != nil {
		return err
	}
	if _, err := s.db.Execute(ctx, `
		CREATE TABLE IF NOT EXISTS queue_items (
			id INTEGER PRIMARY KEY,
			song_mid TEXT NOT NULL,
			title TEXT NOT NULL,
			artist TEXT NOT NULL,
			album TEXT NOT NULL,
			album_mid TEXT NOT NULL,
			duration TEXT NOT NULL,
			song_quality TEXT NOT NULL,
			vip INTEGER NOT NULL,
			status INTEGER NOT NULL,
			progress REAL NOT NULL,
			quality TEXT NOT NULL,
			error TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		)`); err != nil {
		return err
	}
	_, err := s.db.Execute(ctx,
		`CREATE INDEX IF NOT EXISTS idx_queue_items_created_at ON queue_items(created_at)`)
	return err
}

func (s *Store) Add(ctx context.Context, record Record) (int64, error) {
	if record.CompletedAt.IsZero() {
		record.CompletedAt = time.Now()
	}
	if record.Status == "" {
		record.Status = StatusDone
	}
	result, err := s.db.Execute(ctx, `
		INSERT INTO downloads
			(song_mid, title, artist, album, album_mid, quality, file_path, file_size, status, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.SongMID,
		record.Title,
		record.Artist,
		record.Album,
		record.AlbumMID,
		record.Quality,
		record.FilePath,
		record.FileSize,
		record.Status,
		record.CompletedAt.Unix(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertID, nil
}

func (s *Store) Recent(ctx context.Context, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, song_mid, title, artist, album, album_mid, quality, file_path, file_size, status, completed_at
		FROM downloads
		ORDER BY completed_at DESC, id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(rows.Values))
	for _, row := range rows.Values {
		if len(row) < 11 {
			continue
		}
		records = append(records, Record{
			ID:          int64Value(row[0]),
			SongMID:     stringValue(row[1]),
			Title:       stringValue(row[2]),
			Artist:      stringValue(row[3]),
			Album:       stringValue(row[4]),
			AlbumMID:    stringValue(row[5]),
			Quality:     stringValue(row[6]),
			FilePath:    stringValue(row[7]),
			FileSize:    int64Value(row[8]),
			Status:      stringValue(row[9]),
			CompletedAt: time.Unix(int64Value(row[10]), 0).Local(),
		})
	}
	return records, nil
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	rows, err := s.db.Query(ctx, `SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM downloads`)
	if err != nil {
		return Stats{}, err
	}
	if len(rows.Values) == 0 || len(rows.Values[0]) < 2 {
		return Stats{}, nil
	}
	return Stats{
		Count: int(int64Value(rows.Values[0][0])),
		Bytes: int64Value(rows.Values[0][1]),
	}, nil
}

func (s *Store) Clear(ctx context.Context) error {
	_, err := s.db.Execute(ctx, `DELETE FROM downloads`)
	return err
}

func (s *Store) Prune(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.Execute(ctx, `DELETE FROM downloads WHERE completed_at < ?`, before.Unix())
	if err != nil {
		return 0, err
	}
	return result.Changes, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func int64Value(value any) int64 {
	switch number := value.(type) {
	case int64:
		return number
	case int:
		return int64(number)
	case float64:
		return int64(number)
	default:
		return 0
	}
}
