package db

import (
	"context"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Add(ctx context.Context, comics core.Comics) error {
	query := `
			INSERT INTO comics (id, url, title, description, alt)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) 
			DO UPDATE SET 
				url = EXCLUDED.url,
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				alt = EXCLUDED.alt
	`
	_, err := db.conn.ExecContext(ctx, query, comics.ID, comics.URL, comics.Title, comics.Description, comics.Alt)
	if err != nil {
		db.log.Error("db.add error", "error", err)
		return err
	}
	return nil
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	query := `
		SELECT 
			COUNT(*) as comics_fetched,
			COALESCE(SUM(
				CASE 
					WHEN jsonb_typeof(description) = 'object' THEN 
						(SELECT SUM((value::text)::int) FROM jsonb_each_text(description))
					ELSE 0
				END
			), 0) as words_total,
			(SELECT COUNT(DISTINCT key) 
			FROM comics, jsonb_each_text(description) 
			WHERE jsonb_typeof(description) = 'object'
			) as words_unique
		FROM comics
	`
	var res struct {
		WordsTotal    int `db:"words_total"`
		WordsUnique   int `db:"words_unique"`
		ComicsFetched int `db:"comics_fetched"`
	}
	if err := db.conn.GetContext(ctx, &res, query); err != nil {
		return core.DBStats{}, err
	}
	return core.DBStats{
		WordsTotal:    res.WordsTotal,
		WordsUnique:   res.WordsUnique,
		ComicsFetched: res.ComicsFetched,
	}, nil
}

func (db *DB) IDs(ctx context.Context) ([]int, error) {
	var ids []int
	query := "SELECT id FROM comics ORDER BY id"
	if err := db.conn.SelectContext(ctx, &ids, query); err != nil {
		db.log.Error("failed to get comics IDs", "error", err)
		return nil, err
	}
	return ids, nil
}

func (db *DB) Drop(ctx context.Context) error {
	if _, err := db.conn.ExecContext(ctx, "TRUNCATE TABLE comics"); err != nil {
		db.log.Error("failed to drop comics table", "error", err)
		return err
	}
	db.log.Info("comics table dropped successfully")
	return nil
}
