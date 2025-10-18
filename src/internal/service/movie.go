package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	// Validate required fields
	if req.Title == "" {
		return nil, errors.New(422, "UNPROCESSABLE_ENTITY", "title is required")
	}
	if req.Genre == "" {
		return nil, errors.New(422, "UNPROCESSABLE_ENTITY", "genre is required")
	}

	// Parse release date
	releaseDate, err := time.Parse("2006-01-02", req.ReleaseDate)
	if err != nil {
		return nil, errors.New(422, "UNPROCESSABLE_ENTITY", fmt.Sprintf("invalid release_date format, expected YYYY-MM-DD: %v", err))
	}

	// Convert proto request to biz model
	bizReq := &biz.CreateMovieRequest{
		Title:       req.Title,
		Genre:       req.Genre,
		ReleaseDate: releaseDate,
	}

	if req.Distributor != nil {
		bizReq.Distributor = req.Distributor
	}
	if req.Budget != nil {
		bizReq.Budget = req.Budget
	}
	if req.MpaRating != nil {
		bizReq.MPARating = req.MpaRating
	}

	// Call business logic
	movie, err := s.movieUC.CreateMovie(ctx, bizReq)
	if err != nil {
		return nil, err
	}

	// Convert biz model to proto response
	return s.movieToProto(movie), nil
}

// ListMovies implements movie listing
func (s *MovieService) ListMovies(ctx context.Context, req *v1.ListMoviesRequest) (*v1.ListMoviesReply, error) {
	// Convert proto request to biz query
	query := &biz.MovieListQuery{
		Limit: 10, // Default limit
	}

	if req.Q != nil {
		query.Q = req.Q
	}
	if req.Year != nil {
		query.Year = req.Year
	}
	if req.Genre != nil {
		query.Genre = req.Genre
	}
	if req.Distributor != nil {
		query.Distributor = req.Distributor
	}
	if req.Budget != nil {
		query.Budget = req.Budget
	}
	if req.MpaRating != nil {
		query.MPARating = req.MpaRating
	}
	if req.Limit != nil {
		query.Limit = *req.Limit
	}
	if req.Cursor != nil {
		query.Cursor = req.Cursor
	}

	// Call business logic
	page, err := s.movieUC.ListMovies(ctx, query)
	if err != nil {
		return nil, err
	}

	// Convert to proto response
	reply := &v1.ListMoviesReply{
		Items: make([]*v1.MovieItem, 0, len(page.Items)),
	}

	for _, movie := range page.Items {
		reply.Items = append(reply.Items, s.movieItemToProto(movie))
	}

	if page.NextCursor != nil {
		reply.NextCursor = page.NextCursor
	}

	return reply, nil
}

// SubmitRating implements rating submission
func (s *MovieService) SubmitRating(ctx context.Context, req *v1.SubmitRatingRequest) (*v1.SubmitRatingReply, error) {
	// Extract rater ID from context (set by middleware)
	raterID, ok := ctx.Value("rater_id").(string)
	if !ok || raterID == "" {
		return nil, errors.Unauthorized("UNAUTHORIZED", "missing X-Rater-Id header")
	}

	// Call business logic
	rating, isNew, err := s.ratingUC.SubmitRating(ctx, req.Title, raterID, req.Rating)
	if err != nil {
		// Convert "not found" errors to 404
		if err.Error() == "movie not found: movie not found: record not found" ||
			err.Error() == "movie not found: record not found" {
			return nil, errors.NotFound("NOT_FOUND", "movie not found")
		}
		// Convert validation errors to 422
		if err.Error() == "invalid rating value: must be between 0.5 and 5.0 with 0.5 step" {
			return nil, errors.New(422, "INVALID_RATING", err.Error())
		}
		return nil, err
	}

	// Store is_new in context for HTTP status code handling
	ctx = context.WithValue(ctx, "is_new_rating", isNew)

	return &v1.SubmitRatingReply{
		MovieTitle: rating.MovieTitle,
		RaterId:    rating.RaterID,
		Rating:     rating.Rating,
	}, nil
}

// GetRating implements rating aggregation
func (s *MovieService) GetRating(ctx context.Context, req *v1.GetRatingRequest) (*v1.GetRatingReply, error) {
	// Call business logic
	agg, err := s.ratingUC.GetRatingAggregate(ctx, req.Title)
	if err != nil {
		// Convert "not found" errors to 404
		if err.Error() == "movie not found: movie not found: record not found" ||
			err.Error() == "movie not found: record not found" {
			return nil, errors.NotFound("NOT_FOUND", "movie not found")
		}
		return nil, err
	}

	return &v1.GetRatingReply{
		Average: agg.Average,
		Count:   agg.Count,
	}, nil
}

// HealthCheck implements health check
func (s *MovieService) HealthCheck(ctx context.Context, req *v1.HealthCheckRequest) (*v1.HealthCheckReply, error) {
	return &v1.HealthCheckReply{
		Status: "ok",
	}, nil
}

// Helper functions

func (s *MovieService) movieToProto(movie *biz.Movie) *v1.CreateMovieReply {
	reply := &v1.CreateMovieReply{
		Id:          movie.ID,
		Title:       movie.Title,
		ReleaseDate: movie.ReleaseDate.Format("2006-01-02"),
		Genre:       movie.Genre,
	}

	if movie.Distributor != nil {
		reply.Distributor = movie.Distributor
	}
	if movie.Budget != nil {
		reply.Budget = movie.Budget
	}
	if movie.MPARating != nil {
		reply.MpaRating = movie.MPARating
	}

	// BoxOffice: keep as nil if not present (upstream failure)
	// This allows JSON to serialize it as null in create response
	if movie.BoxOffice != nil {
		reply.BoxOffice = &v1.BoxOffice{
			Revenue: &v1.Revenue{
				Worldwide: movie.BoxOffice.Revenue.Worldwide,
			},
			Currency:    movie.BoxOffice.Currency,
			Source:      movie.BoxOffice.Source,
			LastUpdated: timestamppb.New(movie.BoxOffice.LastUpdated),
		}
		if movie.BoxOffice.Revenue.OpeningWeekendUSA != nil {
			reply.BoxOffice.Revenue.OpeningWeekendUsa = movie.BoxOffice.Revenue.OpeningWeekendUSA
		}
	}
	// Note: Do NOT set empty BoxOffice here - let it be nil for null serialization

	return reply
}

func (s *MovieService) movieItemToProto(movie *biz.Movie) *v1.MovieItem {
	item := &v1.MovieItem{
		Id:          movie.ID,
		Title:       movie.Title,
		ReleaseDate: movie.ReleaseDate.Format("2006-01-02"),
		Genre:       movie.Genre,
	}

	if movie.Distributor != nil {
		item.Distributor = movie.Distributor
	}
	if movie.Budget != nil {
		item.Budget = movie.Budget
	}
	if movie.MPARating != nil {
		item.MpaRating = movie.MPARating
	}

	// Always include boxOffice field (even if nil) to satisfy API contract
	if movie.BoxOffice != nil {
		item.BoxOffice = &v1.BoxOffice{
			Revenue: &v1.Revenue{
				Worldwide: movie.BoxOffice.Revenue.Worldwide,
			},
			Currency:    movie.BoxOffice.Currency,
			Source:      movie.BoxOffice.Source,
			LastUpdated: timestamppb.New(movie.BoxOffice.LastUpdated),
		}
		if movie.BoxOffice.Revenue.OpeningWeekendUSA != nil {
			item.BoxOffice.Revenue.OpeningWeekendUsa = movie.BoxOffice.Revenue.OpeningWeekendUSA
		}
	} else {
		// Set empty BoxOffice to ensure field appears in JSON (even as null)
		item.BoxOffice = &v1.BoxOffice{}
	}

	return item
}
