package main

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type DBConfig struct {
	UrlString string `json:"url_string"`
}

func NewDBConfig(urlString string) *DBConfig {
	return &DBConfig{
		UrlString: urlString,
	}
}

func IsChatRoomExists(ctx context.Context, tx pgx.Tx, roomID string) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM chat WHERE room_id = $1)`

	var exists bool
	err := tx.QueryRow(ctx, q, roomID).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func CreateChat(ctx context.Context, tx pgx.Tx, wimr *WebhookIncomingMessageRequest) error {
	q := `INSERT INTO chat(room_id, data) VALUES ( $1, $2 )`

	_, err := tx.Exec(ctx, q, wimr.RoomID, wimr)

	if err != nil {
		return err
	}

	return nil
}

func UpdateChat(ctx context.Context, tx pgx.Tx, wimr *WebhookIncomingMessageRequest) error {
	q := `UPDATE chat SET status = $1 WHERE room_id = $2`

	_, err := tx.Exec(ctx, q, "SERVED", wimr.RoomID)

	if err != nil {
		return err
	}

	return nil
}
