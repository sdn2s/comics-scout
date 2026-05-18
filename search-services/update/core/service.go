package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	publisher   EventPublisher
	concurrency int

	statusMu sync.Mutex
	status   ServiceStatus
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, publisher EventPublisher, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		publisher:   publisher,
		concurrency: concurrency,
		status:      StatusIdle,
	}, nil
}

func (s *Service) Update(ctx context.Context) (err error) {
	if !s.setStatusRunning() {
		return ErrAlreadyExists
	}
	defer s.setStatusIdle()

	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return err
	}

	existingIDs, err := s.db.IDs(ctx)
	if err != nil {
		return err
	}

	known := make(map[int]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		known[id] = struct{}{}
	}

	idsCh := make(chan int)
	errCh := make(chan error, s.concurrency)

	var wg sync.WaitGroup
	wg.Add(s.concurrency)

	for i := 0; i < s.concurrency; i++ {
		go func() {
			defer wg.Done()
			for id := range idsCh {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				default:
				}

				info, getErr := s.xkcd.Get(ctx, id)
				if getErr != nil {
					if getErr == ErrNotFound {
						continue
					}
					errCh <- getErr
					continue
				}

				words, normErr := s.words.Norm(ctx, info.Description)
				if normErr != nil {
					errCh <- normErr
					continue
				}

				addErr := s.db.Add(ctx, Comics{
					ID:    info.ID,
					URL:   info.URL,
					Words: words,
				})
				if addErr != nil {
					errCh <- addErr
					continue
				}
			}
		}()
	}

	for id := 1; id <= lastID; id++ {
		if _, ok := known[id]; ok {
			continue
		}

		select {
		case <-ctx.Done():
			close(idsCh)
			wg.Wait()
			close(errCh)
			return ctx.Err()
		case idsCh <- id:
		}
	}
	close(idsCh)

	wg.Wait()
	close(errCh)

	for workerErr := range errCh {
		if workerErr != nil && workerErr != context.Canceled {
			return workerErr
		}
	}

	if s.publisher != nil {
		if pubErr := s.publisher.Publish(ctx, "xkcd.db.updated", []byte("updated")); pubErr != nil {
			return pubErr
		}
	}

	return nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	stats, err := s.db.Stats(ctx)
	if err != nil {
		return ServiceStats{}, err
	}

	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return ServiceStats{}, err
	}

	return ServiceStats{
		DBStats:     stats,
		ComicsTotal: lastID,
	}, nil
}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	return s.status
}

func (s *Service) Drop(ctx context.Context) error {
	if !s.setStatusRunning() {
		return ErrAlreadyExists
	}
	defer s.setStatusIdle()

	if err := s.db.Drop(ctx); err != nil {
		return err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, "xkcd.db.dropped", []byte("dropped")); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) setStatusRunning() bool {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	if s.status == StatusRunning {
		return false
	}

	s.status = StatusRunning
	return true
}

func (s *Service) setStatusIdle() {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	s.status = StatusIdle
}
