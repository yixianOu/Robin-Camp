package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"src/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type movieRepo struct {
	data *Data
	log  *log.Helper
}

// NewMovieRepo creates a new movie repository
func NewMovieRepo(data *Data, logger log.Logger) biz.MovieRepo {
	return &movieRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *movieRepo) CreateMovie(ctx context.Context, movie *biz.Movie) error {
	// Convert biz.Movie to data.Movie
	dbMovie := r.bizToModel(movie)

	// Save to database
	if err := r.data.db.WithContext(ctx).Create(dbMovie).Error; err != nil {
		return fmt.Errorf("failed to create movie: %w", err)
	}

	// Invalidate cache if Redis is available
	if r.data.rdb != nil {
		cacheKey := fmt.Sprintf("movie:%s", movie.Title)
		r.data.rdb.Del(ctx, cacheKey)
	}

	return nil
}

func (r *movieRepo) GetMovieByTitle(ctx context.Context, title string) (*biz.Movie, error) {
	// Try cache first if Redis is available
	if r.data.rdb != nil {
		cacheKey := fmt.Sprintf("movie:%s", title)
		cached, err := r.data.rdb.Get(ctx, cacheKey).Result()
		if err == nil {
			var movie biz.Movie
			if err := json.Unmarshal([]byte(cached), &movie); err == nil {
				r.log.Debugf("cache hit for movie: %s", title)
				return &movie, nil
			}
		}
	}

	// Query from database
	var dbMovie Movie
	if err := r.data.db.WithContext(ctx).Where("title = ?", title).First(&dbMovie).Error; err != nil {
		return nil, fmt.Errorf("movie not found: %w", err)
	}

	// Convert to biz model
	movie := r.modelToBiz(&dbMovie)

	// Cache result if Redis is available
	if r.data.rdb != nil {
		cacheKey := fmt.Sprintf("movie:%s", title)
		if data, err := json.Marshal(movie); err == nil {
			r.data.rdb.Set(ctx, cacheKey, data, 15*time.Minute)
		}
	}

	return movie, nil
}

func (r *movieRepo) ListMovies(ctx context.Context, query *biz.MovieListQuery) (*biz.MoviePage, error) {
	// TODO: Implement list with filters and pagination
	return &biz.MoviePage{
		Items:      []*biz.Movie{},
		NextCursor: nil,
	}, nil
}

func (r *movieRepo) UpdateMovie(ctx context.Context, movie *biz.Movie) error {
	dbMovie := r.bizToModel(movie)

	if err := r.data.db.WithContext(ctx).Save(dbMovie).Error; err != nil {
		return fmt.Errorf("failed to update movie: %w", err)
	}

	// Invalidate cache
	if r.data.rdb != nil {
		cacheKey := fmt.Sprintf("movie:%s", movie.Title)
		r.data.rdb.Del(ctx, cacheKey)
	}

	return nil
}

// Helper: Convert biz.Movie to data.Movie
func (r *movieRepo) bizToModel(biz *biz.Movie) *Movie {
	m := &Movie{
		ID:          biz.ID,
		Title:       biz.Title,
		ReleaseDate: biz.ReleaseDate,
		Genre:       biz.Genre,
		Distributor: biz.Distributor,
		Budget:      biz.Budget,
		MPARating:   biz.MPARating,
	}

	if biz.BoxOffice != nil {
		worldwide := biz.BoxOffice.Revenue.Worldwide
		m.BoxOfficeWorldwide = &worldwide

		if biz.BoxOffice.Revenue.OpeningWeekendUSA != nil {
			m.BoxOfficeOpeningUSA = biz.BoxOffice.Revenue.OpeningWeekendUSA
		}

		currency := biz.BoxOffice.Currency
		m.BoxOfficeCurrency = &currency

		source := biz.BoxOffice.Source
		m.BoxOfficeSource = &source

		lastUpdated := biz.BoxOffice.LastUpdated
		m.BoxOfficeLastUpdated = &lastUpdated
	}

	return m
}

// Helper: Convert data.Movie to biz.Movie
func (r *movieRepo) modelToBiz(m *Movie) *biz.Movie {
	movie := &biz.Movie{
		ID:          m.ID,
		Title:       m.Title,
		ReleaseDate: m.ReleaseDate,
		Genre:       m.Genre,
		Distributor: m.Distributor,
		Budget:      m.Budget,
		MPARating:   m.MPARating,
	}

	if m.BoxOfficeWorldwide != nil {
		movie.BoxOffice = &biz.BoxOffice{
			Revenue: biz.Revenue{
				Worldwide:         *m.BoxOfficeWorldwide,
				OpeningWeekendUSA: m.BoxOfficeOpeningUSA,
			},
		}

		if m.BoxOfficeCurrency != nil {
			movie.BoxOffice.Currency = *m.BoxOfficeCurrency
		}
		if m.BoxOfficeSource != nil {
			movie.BoxOffice.Source = *m.BoxOfficeSource
		}
		if m.BoxOfficeLastUpdated != nil {
			movie.BoxOffice.LastUpdated = *m.BoxOfficeLastUpdated
		}
	}

	return movie
}
