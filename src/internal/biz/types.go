package biz

import (
	"context"
	"time"
)

// Movie domain model
type Movie struct {
	ID          string
	Title       string
	ReleaseDate time.Time
	Genre       string
	Distributor *string
	Budget      *int64
	MPARating   *string
	BoxOffice   *BoxOffice
}

// BoxOffice domain model
type BoxOffice struct {
	Revenue     Revenue
	Currency    string
	Source      string
	LastUpdated time.Time
}

// Revenue domain model
type Revenue struct {
	Worldwide         int64
	OpeningWeekendUSA *int64
}

// CreateMovieRequest domain model
type CreateMovieRequest struct {
	Title       string
	Genre       string
	ReleaseDate time.Time
	Distributor *string
	Budget      *int64
	MPARating   *string
}

// Rating domain model
type Rating struct {
	MovieTitle string
	RaterID    string
	Rating     float64
}

// RatingAggregate domain model
type RatingAggregate struct {
	Average float64
	Count   int32
}

// MovieListQuery domain model
type MovieListQuery struct {
	Q           *string
	Year        *int32
	Genre       *string
	Distributor *string
	Budget      *int64
	MPARating   *string
	Limit       int32
	Cursor      *string
}

// MoviePage domain model
type MoviePage struct {
	Items      []*Movie
	NextCursor *string
}

// MovieRepo defines the repository interface for movies
type MovieRepo interface {
	CreateMovie(ctx context.Context, movie *Movie) error
	GetMovieByTitle(ctx context.Context, title string) (*Movie, error)
	ListMovies(ctx context.Context, query *MovieListQuery) (*MoviePage, error)
	UpdateMovie(ctx context.Context, movie *Movie) error
}

// RatingRepo defines the repository interface for ratings
type RatingRepo interface {
	UpsertRating(ctx context.Context, rating *Rating) (isNew bool, err error)
	GetRatingAggregate(ctx context.Context, movieTitle string) (*RatingAggregate, error)
}

// BoxOfficeClient defines the interface for box office API client
type BoxOfficeClient interface {
	GetBoxOffice(ctx context.Context, title string) (*BoxOfficeData, error)
}

// BoxOfficeData represents data from box office API
type BoxOfficeData struct {
	Title       string
	Distributor *string
	ReleaseDate *time.Time
	Budget      *int64
	Revenue     *BoxOfficeRevenue
	MPARating   *string
}

// BoxOfficeRevenue represents revenue data from box office API
type BoxOfficeRevenue struct {
	Worldwide         int64
	OpeningWeekendUSA *int64
}
