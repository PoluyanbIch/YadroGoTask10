package db

import (
	"context"
	"encoding/json"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/search/core"
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

func (db *DB) Read(ctx context.Context) ([]core.DBComic, error) {
	query := `
			SELECT id, url, title, description, alt FROM comics
	`
	var rawComics []struct {
		ID          int             `db:"id"`
		URL         string          `db:"url"`
		Title       json.RawMessage `db:"title"`
		Description json.RawMessage `db:"description"`
		Alt         json.RawMessage `db:"alt"`
	}

	err := db.conn.SelectContext(ctx, &rawComics, query)
	if err != nil {
		db.log.Error("db.read error", "error", err)
		return nil, err
	}

	comics := make([]core.DBComic, len(rawComics))
	for i, raw := range rawComics {
		var descriptionMap map[string]int
		if len(raw.Description) > 0 {
			if err := json.Unmarshal(raw.Description, &descriptionMap); err != nil {
				db.log.Error("json unmarshall error", "error", err)
				return nil, err
			}
		} else {
			descriptionMap = make(map[string]int)
		}
		var altMap map[string]int
		if len(raw.Alt) > 0 {
			if err := json.Unmarshal(raw.Alt, &altMap); err != nil {
				db.log.Error("json unmarshall error", "error", err)
				return nil, err
			}
		} else {
			altMap = make(map[string]int)
		}
		var titleMap map[string]int
		if len(raw.Title) > 0 {
			if err := json.Unmarshal(raw.Title, &titleMap); err != nil {
				db.log.Error("json unmarshall error", "error", err)
				return nil, err
			}
		} else {
			titleMap = make(map[string]int)
		}
		comics[i] = core.DBComic{
			ID:          raw.ID,
			URL:         raw.URL,
			Description: descriptionMap,
			Title:       titleMap,
			Alt:         altMap,
		}
	}
	return comics, nil
}
