package history

import (
	"context"
	"time"
)

type LogRecord struct {
	At      time.Time
	Level   string
	Message string
}

func (s *Store) AddLog(ctx context.Context, record LogRecord) error {
	if record.At.IsZero() {
		record.At = time.Now()
	}
	_, err := s.db.Execute(ctx,
		`INSERT INTO logs (level, message, created_at) VALUES (?, ?, ?)`,
		record.Level, record.Message, record.At.Unix())
	return err
}

func (s *Store) LogsSince(ctx context.Context, since time.Time) ([]LogRecord, error) {
	rows, err := s.db.Query(ctx, `
		SELECT level, message, created_at
		FROM logs
		WHERE created_at >= ?
		ORDER BY created_at ASC, id ASC`, since.Unix())
	if err != nil {
		return nil, err
	}
	records := make([]LogRecord, 0, len(rows.Values))
	for _, row := range rows.Values {
		if len(row) < 3 {
			continue
		}
		records = append(records, LogRecord{
			Level:   stringValue(row[0]),
			Message: stringValue(row[1]),
			At:      time.Unix(int64Value(row[2]), 0).Local(),
		})
	}
	return records, nil
}

func (s *Store) PruneLogs(ctx context.Context, before time.Time) error {
	_, err := s.db.Execute(ctx, `DELETE FROM logs WHERE created_at < ?`, before.Unix())
	return err
}
