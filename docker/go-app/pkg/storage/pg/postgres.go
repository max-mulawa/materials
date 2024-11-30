package pg

import (
	"context"
	"fmt"
	"log/slog"

	"api.com/quick/pkg/messages"
	"api.com/quick/pkg/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func New(conn string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(context.Background(), conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create db pool: %w", err)
	}
	return &PostgresStorage{
		pool: pool,
	}, nil
}

func (s *PostgresStorage) Store(msg messages.Message) error {
	cmdTag, err := s.pool.Exec(context.TODO(), "INSERT INTO messages(id, title, description) "+
		"VALUES ($1, $2, $3) ",
		msg.Id, msg.Title, msg.Body)
	if err != nil {
		return fmt.Errorf("insert failed : %w", err)
	}
	slog.Info("insert executed", "tag", cmdTag)
	return nil
}

func (s *PostgresStorage) Load(id messages.MsgID) (messages.Message, error) {
	row := s.pool.QueryRow(context.TODO(), "SELECT id, title, description FROM messages WHERE id=$1", id)
	msg := messages.Message{}
	err := row.Scan(&msg.Id, &msg.Title, &msg.Body)
	if err == pgx.ErrNoRows {
		return messages.Message{}, storage.ErrNotFound
	} else if err != nil {
		return messages.Message{}, fmt.Errorf("failed to query row: %w", err)
	}

	return msg, nil
}

func (s *PostgresStorage) All() ([]messages.Message, error) {
	cur, err := s.pool.Query(context.TODO(), "SELECT id, title, description FROM messages ORDER BY id DESC")
	if err != nil {
		return []messages.Message{}, fmt.Errorf("query failed: %v", err)
	}
	defer cur.Close()

	res := make([]messages.Message, 0)

	for cur.Next() {
		msg := messages.Message{}
		err := cur.Scan(&msg.Id, &msg.Title, &msg.Body)
		if err != nil {
			return []messages.Message{}, fmt.Errorf("scan failed: %v", err)
		}
		res = append(res, msg)
	}

	return res, nil
}
