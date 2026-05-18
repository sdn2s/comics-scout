package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/update/adapters/words"
)

type Service struct {
	log   *slog.Logger
	db    *sqlx.DB
	words *words.Client
}

func New(log *slog.Logger, db *sqlx.DB, wordsClient *words.Client) *Service {
	return &Service{
		log:   log,
		db:    db,
		words: wordsClient,
	}
}

func (s *Service) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	limit := req.GetLimit()
	if limit <= 0 {
		limit = 10
	}

	normWords, err := s.words.Norm(ctx, req.GetPhrase())
	if err != nil {
		s.log.Error("normalize failed", "error", err)
		return nil, err
	}
	if len(normWords) == 0 {
		return &searchpb.SearchReply{}, nil
	}

	var comics []struct {
		ID    int    `db:"id"`
		URL   string `db:"url"`
		Score int    `db:"score"`
	}

	query := `
SELECT id, url,
       cardinality((SELECT ARRAY(
           SELECT unnest(words) INTERSECT SELECT unnest($1::text[])
       ))) AS score
FROM comics
WHERE words && $1
ORDER BY score DESC, id
LIMIT $2`

	if err := s.db.SelectContext(ctx, &comics, query, pq.Array(normWords), limit); err != nil {
		return nil, fmt.Errorf("query comics: %w", err)
	}

	resp := &searchpb.SearchReply{
		Comics: make([]*searchpb.Comics, 0, len(comics)),
	}
	for _, c := range comics {
		resp.Comics = append(resp.Comics, &searchpb.Comics{
			Id:  int64(c.ID),
			Url: c.URL,
		})
	}

	return resp, nil
}
