package data

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
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
	// Decode cursor to get offset
	offset := 0
	if query.Cursor != nil && *query.Cursor != "" {
		var err error
		offset, err = decodeCursor(*query.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
	}

	// Build query
	db := r.data.db.WithContext(ctx).Model(&Movie{})

	// Apply filters
	if query.Q != nil && *query.Q != "" {
		searchTerm := fmt.Sprintf("%%%s%%", *query.Q)
		db = db.Where("title ILIKE ?", searchTerm)
	}

	if query.Year != nil {
		db = db.Where("EXTRACT(YEAR FROM release_date) = ?", *query.Year)
	}

	if query.Genre != nil {
		db = db.Where("LOWER(genre) = LOWER(?)", *query.Genre)
	}

	if query.Distributor != nil {
		db = db.Where("LOWER(distributor) = LOWER(?)", *query.Distributor)
	}

	if query.Budget != nil {
		db = db.Where("budget <= ?", *query.Budget)
	}

	if query.MPARating != nil {
		db = db.Where("mpa_rating = ?", *query.MPARating)
	}

	// Apply pagination - fetch limit+1 to detect if there are more pages
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	var dbMovies []Movie
	err := db.Offset(offset).Limit(int(limit + 1)).Find(&dbMovies).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list movies: %w", err)
	}

	// Check if there's a next page
	hasMore := len(dbMovies) > int(limit)
	if hasMore {
		dbMovies = dbMovies[:limit] // Remove the extra item
	}

	// Convert to biz models
	movies := make([]*biz.Movie, 0, len(dbMovies))
	for i := range dbMovies {
		movies = append(movies, r.modelToBiz(&dbMovies[i]))
	}

	// Prepare result
	result := &biz.MoviePage{
		Items: movies,
	}

	// Generate next cursor if there are more pages
	if hasMore {
		nextOffset := offset + int(limit)
		nextCursor := encodeCursor(nextOffset)
		result.NextCursor = &nextCursor
	}

	return result, nil
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

// encodeCursor encodes an offset into a base64 cursor string
func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

// decodeCursor decodes a base64 cursor string back to an offset
func decodeCursor(cursor string) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	offset, err := strconv.Atoi(string(decoded))
	if err != nil {
		return 0, fmt.Errorf("invalid cursor format: %w", err)
	}

	return offset, nil
}
