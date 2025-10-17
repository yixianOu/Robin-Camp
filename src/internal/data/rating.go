package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"src/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm/clause"
)

type ratingRepo struct {
	data *Data
	log  *log.Helper
}

// NewRatingRepo creates a new rating repository
func NewRatingRepo(data *Data, logger log.Logger) biz.RatingRepo {
	return &ratingRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *ratingRepo) UpsertRating(ctx context.Context, rating *biz.Rating) (bool, error) {
	dbRating := &Rating{
		MovieTitle: rating.MovieTitle,
		RaterID:    rating.RaterID,
		Rating:     rating.Rating,
	}

	// Use GORM's ON CONFLICT clause for upsert
	result := r.data.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "movie_title"}, {Name: "rater_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"rating", "updated_at"}),
	}).Create(dbRating)

	if result.Error != nil {
		return false, fmt.Errorf("failed to upsert rating: %w", result.Error)
	}

	// Determine if it was an insert or update
	isNew := result.RowsAffected > 0

	// Invalidate rating aggregate cache
	if r.data.rdb != nil {
		cacheKey := fmt.Sprintf("rating:agg:%s", rating.MovieTitle)
		r.data.rdb.Del(ctx, cacheKey)

		// Update Redis ZSet for rankings
		r.updateRankings(ctx, rating.MovieTitle)
	}

	return isNew, nil
}

func (r *ratingRepo) GetRatingAggregate(ctx context.Context, movieTitle string) (*biz.RatingAggregate, error) {
	// Try cache first if Redis is available
	if r.data.rdb != nil {
		cacheKey := fmt.Sprintf("rating:agg:%s", movieTitle)
		cached, err := r.data.rdb.Get(ctx, cacheKey).Result()
		if err == nil {
			var agg biz.RatingAggregate
			if err := json.Unmarshal([]byte(cached), &agg); err == nil {
				r.log.Debugf("cache hit for rating aggregate: %s", movieTitle)
				return &agg, nil
			}
		}
	}

	// Query from database
	var result struct {
		Average float64
		Count   int32
	}

	err := r.data.db.WithContext(ctx).
		Model(&Rating{}).
		Select("ROUND(AVG(rating)::numeric, 1) as average, COUNT(*) as count").
		Where("movie_title = ?", movieTitle).
		Scan(&result).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get rating aggregate: %w", err)
	}

	agg := &biz.RatingAggregate{
		Average: result.Average,
		Count:   result.Count,
	}

	// Cache result if Redis is available
	if r.data.rdb != nil {
		cacheKey := fmt.Sprintf("rating:agg:%s", movieTitle)
		if data, err := json.Marshal(agg); err == nil {
			r.data.rdb.Set(ctx, cacheKey, data, 15*time.Minute)
		}
	}

	return agg, nil
}

// updateRankings updates Redis ZSet rankings
func (r *ratingRepo) updateRankings(ctx context.Context, movieTitle string) {
	if r.data.rdb == nil {
		return
	}

	// Get movie aggregate
	var result struct {
		Average float64
		Count   int32
	}

	err := r.data.db.WithContext(ctx).
		Model(&Rating{}).
		Select("AVG(rating) as average, COUNT(*) as count").
		Where("movie_title = ?", movieTitle).
		Scan(&result).Error

	if err != nil {
		r.log.Warnf("failed to get aggregate for ranking update: %v", err)
		return
	}

	// Update popular movies ranking (by rating count)
	r.data.rdb.ZAdd(ctx, "rank:movies:popular", redis.Z{
		Score:  float64(result.Count),
		Member: movieTitle,
	})

	// Update top-rated movies ranking (by average rating)
	if result.Count > 0 {
		r.data.rdb.ZAdd(ctx, "rank:movies:top", redis.Z{
			Score:  result.Average,
			Member: movieTitle,
		})
	}
}
