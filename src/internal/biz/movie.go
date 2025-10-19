package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// MovieUseCase handles movie-related business logic
type MovieUseCase struct {
	repo            MovieRepo
	boxOfficeClient BoxOfficeClient
	log             *log.Helper
}

// NewMovieUseCase creates a new MovieUseCase instance
func NewMovieUseCase(repo MovieRepo, boxOfficeClient BoxOfficeClient, logger log.Logger) *MovieUseCase {
	return &MovieUseCase{
		repo:            repo,
		boxOfficeClient: boxOfficeClient,
		log:             log.NewHelper(logger),
	}
}

// CreateMovie creates a new movie and fetches box office data
func (uc *MovieUseCase) CreateMovie(ctx context.Context, req *CreateMovieRequest) (*Movie, error) {
	// Generate movie ID (UUID v7: time-ordered, distributed-friendly)
	movieID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate movie ID: %w", err)
	}

	// Create base movie object
	movie := &Movie{
		ID:          movieID.String(),
		Title:       req.Title,
		ReleaseDate: req.ReleaseDate,
		Genre:       req.Genre,
		Distributor: req.Distributor,
		Budget:      req.Budget,
		MPARating:   req.MPARating,
	}

	// Try to fetch box office data (non-blocking on failure)
	boxOfficeData, err := uc.boxOfficeClient.GetBoxOffice(ctx, req.Title)
	if err != nil {
		uc.log.Warnf("Failed to fetch box office data for movie '%s': %v", req.Title, err)
		// Continue with creation, boxOffice will be nil
	} else if boxOfficeData != nil {
		// Merge box office data, but user-provided values take precedence
		if movie.Distributor == nil && boxOfficeData.Distributor != nil {
			movie.Distributor = boxOfficeData.Distributor
		}
		if movie.Budget == nil && boxOfficeData.Budget != nil {
			movie.Budget = boxOfficeData.Budget
		}
		if movie.MPARating == nil && boxOfficeData.MPARating != nil {
			movie.MPARating = boxOfficeData.MPARating
		}

		// Set box office data
		if boxOfficeData.Revenue != nil {
			movie.BoxOffice = &BoxOffice{
				Revenue: Revenue{
					Worldwide:         boxOfficeData.Revenue.Worldwide,
					OpeningWeekendUSA: boxOfficeData.Revenue.OpeningWeekendUSA,
				},
				Currency:    "USD", // Default currency
				Source:      "BoxOfficeAPI",
				LastUpdated: time.Now().UTC(),
			}
		}
	}

	// Save to database
	if err := uc.repo.CreateMovie(ctx, movie); err != nil {
		return nil, fmt.Errorf("failed to create movie: %w", err)
	}

	return movie, nil
}

// GetMovieByTitle retrieves a movie by its title
func (uc *MovieUseCase) GetMovieByTitle(ctx context.Context, title string) (*Movie, error) {
	movie, err := uc.repo.GetMovieByTitle(ctx, title)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}
	return movie, nil
}

// ListMovies retrieves a paginated list of movies based on filters
func (uc *MovieUseCase) ListMovies(ctx context.Context, query *MovieListQuery) (*MoviePage, error) {
	page, err := uc.repo.ListMovies(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list movies: %w", err)
	}
	return page, nil
}
