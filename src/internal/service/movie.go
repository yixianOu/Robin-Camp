package service

import (
	"context"

	v1 "src/api/movie/v1"
	"src/internal/biz"
)

// MovieService implements the MovieService API
type MovieService struct {
	v1.UnimplementedMovieServiceServer

	movieUC  *biz.MovieUseCase
	ratingUC *biz.RatingUseCase
}

// NewMovieService creates a new MovieService
func NewMovieService(movieUC *biz.MovieUseCase, ratingUC *biz.RatingUseCase) *MovieService {
	return &MovieService{
		movieUC:  movieUC,
		ratingUC: ratingUC,
	}
}

// CreateMovie implements movie creation
func (s *MovieService) CreateMovie(ctx context.Context, req *v1.CreateMovieRequest) (*v1.CreateMovieReply, error) {
	// TODO: Implement
	return &v1.CreateMovieReply{}, nil
}

// ListMovies implements movie listing
func (s *MovieService) ListMovies(ctx context.Context, req *v1.ListMoviesRequest) (*v1.ListMoviesReply, error) {
	// TODO: Implement
	return &v1.ListMoviesReply{}, nil
}

// SubmitRating implements rating submission
func (s *MovieService) SubmitRating(ctx context.Context, req *v1.SubmitRatingRequest) (*v1.SubmitRatingReply, error) {
	// TODO: Implement
	return &v1.SubmitRatingReply{}, nil
}

// GetRating implements rating aggregation
func (s *MovieService) GetRating(ctx context.Context, req *v1.GetRatingRequest) (*v1.GetRatingReply, error) {
	// TODO: Implement
	return &v1.GetRatingReply{}, nil
}

// HealthCheck implements health check
func (s *MovieService) HealthCheck(ctx context.Context, req *v1.HealthCheckRequest) (*v1.HealthCheckReply, error) {
	return &v1.HealthCheckReply{
		Status: "ok",
	}, nil
}
