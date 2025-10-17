package biz

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
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
		return nil, false, fmt.Errorf("movie not found: %w", err)
	}

	// Validate rating value
	if !isValidRating(ratingValue) {
		return nil, false, fmt.Errorf("invalid rating value: must be between 0.5 and 5.0 with 0.5 step")
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
		return nil, fmt.Errorf("movie not found: %w", err)
	}

	// Get aggregated rating
	aggregate, err := uc.ratingRepo.GetRatingAggregate(ctx, movieTitle)
	if err != nil {
		return nil, fmt.Errorf("failed to get rating aggregate: %w", err)
	}

	return aggregate, nil
}

// isValidRating checks if the rating value is valid (0.5 to 5.0 with 0.5 step)
func isValidRating(rating float64) bool {
	validRatings := []float64{0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0}
	for _, valid := range validRatings {
		if rating == valid {
			return true
		}
	}
	return false
}
