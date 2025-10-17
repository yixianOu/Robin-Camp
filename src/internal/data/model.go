package data

import (
	"time"

	"gorm.io/gorm"
)

// Movie represents the movies table
type Movie struct {
	ID          string    `gorm:"primaryKey;size:64"`
	Title       string    `gorm:"uniqueIndex;not null;size:255"`
	ReleaseDate time.Time `gorm:"not null;type:date"`
	Genre       string    `gorm:"not null;size:100;index:idx_movies_genre,expression:LOWER(genre)"`
	Distributor *string   `gorm:"size:255;index:idx_movies_distributor,expression:LOWER(distributor)"`
	Budget      *int64    `gorm:"index:idx_movies_budget"`
	MPARating   *string   `gorm:"column:mpa_rating;size:10;index:idx_movies_mpa_rating"`

	// Box Office fields (nullable)
	BoxOfficeWorldwide   *int64     `gorm:"column:box_office_worldwide"`
	BoxOfficeOpeningUSA  *int64     `gorm:"column:box_office_opening_usa"`
	BoxOfficeCurrency    *string    `gorm:"column:box_office_currency;size:10"`
	BoxOfficeSource      *string    `gorm:"column:box_office_source;size:100"`
	BoxOfficeLastUpdated *time.Time `gorm:"column:box_office_last_updated;type:timestamptz"`

	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName overrides the table name
func (Movie) TableName() string {
	return "movies"
}

// Rating represents the ratings table
type Rating struct {
	ID         uint      `gorm:"primaryKey"`
	MovieTitle string    `gorm:"not null;size:255;uniqueIndex:uq_rating_movie_rater;index:idx_ratings_movie_title"`
	RaterID    string    `gorm:"not null;size:100;uniqueIndex:uq_rating_movie_rater"`
	Rating     float64   `gorm:"not null;type:decimal(2,1);check:rating >= 0.5 AND rating <= 5.0 AND MOD(rating * 10, 5) = 0"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`

	// Foreign key
	Movie Movie `gorm:"foreignKey:MovieTitle;references:Title;constraint:OnDelete:CASCADE"`
}

// TableName overrides the table name
func (Rating) TableName() string {
	return "ratings"
}

// RatingAggregate represents the aggregated rating result
type RatingAggregate struct {
	Average float64
	Count   int32
}
