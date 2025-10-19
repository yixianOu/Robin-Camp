package biz

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// Custom errors
var (
	ErrMovieNotFound = errors.New("movie not found")
)

// RatingUseCase handles rating-related business logic
type RatingUseCase struct {
	movieRepo  MovieRepo
	ratingRepo RatingRepo
	log        *log.Helper
}

// NewRatingUseCase creates a new RatingUseCase instance
func NewRatingUseCase(movieRepo MovieRepo, ratingRepo RatingRepo, logger log.Logger) *RatingUseCase {
	return &RatingUseCase{
		movieRepo:  movieRepo,
		ratingRepo: ratingRepo,
		log:        log.NewHelper(logger),
	}
}

// SubmitRating submits or updates a rating for a movie (Upsert)
func (uc *RatingUseCase) SubmitRating(ctx context.Context, movieTitle, raterID string, ratingValue float64) (*Rating, bool, error) {
	// Check if movie exists
	_, err := uc.movieRepo.GetMovieByTitle(ctx, movieTitle)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrMovieNotFound, err)
	}

	// Create rating object
	rating := &Rating{
		MovieTitle: movieTitle,
		RaterID:    raterID,
		Rating:     ratingValue,
	}

	// Upsert rating
	isNew, err := uc.ratingRepo.UpsertRating(ctx, rating)
	if err != nil {
		return nil, false, fmt.Errorf("failed to upsert rating: %w", err)
	}

	return rating, isNew, nil
}

// GetRatingAggregate retrieves aggregated rating for a movie
func (uc *RatingUseCase) GetRatingAggregate(ctx context.Context, movieTitle string) (*RatingAggregate, error) {
	// Check if movie exists
	_, err := uc.movieRepo.GetMovieByTitle(ctx, movieTitle)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMovieNotFound, err)
	}

	// Get aggregated rating
	aggregate, err := uc.ratingRepo.GetRatingAggregate(ctx, movieTitle)
	if err != nil {
		return nil, fmt.Errorf("failed to get rating aggregate: %w", err)
	}

	return aggregate, nil
}
