-- Initialize database schema for Movie Rating API

-- Create movies table
CREATE TABLE IF NOT EXISTS movies (
    id VARCHAR(64) PRIMARY KEY,
    title VARCHAR(255) NOT NULL UNIQUE,
    release_date DATE NOT NULL,
    genre VARCHAR(100) NOT NULL,
    distributor VARCHAR(255),
    budget BIGINT,
    mpa_rating VARCHAR(10),
    
    -- Box Office data (nullable if upstream fails)
    box_office_worldwide BIGINT,
    box_office_opening_usa BIGINT,
    box_office_currency VARCHAR(10),
    box_office_source VARCHAR(100),
    box_office_last_updated TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_movies_title ON movies(title);
CREATE INDEX IF NOT EXISTS idx_movies_year ON movies(EXTRACT(YEAR FROM release_date));
CREATE INDEX IF NOT EXISTS idx_movies_genre ON movies(LOWER(genre));
CREATE INDEX IF NOT EXISTS idx_movies_distributor ON movies(LOWER(distributor));
CREATE INDEX IF NOT EXISTS idx_movies_budget ON movies(budget);
CREATE INDEX IF NOT EXISTS idx_movies_mpa_rating ON movies(mpa_rating);

-- Create ratings table
CREATE TABLE IF NOT EXISTS ratings (
    id SERIAL PRIMARY KEY,
    movie_title VARCHAR(255) NOT NULL,
    rater_id VARCHAR(100) NOT NULL,
    rating DECIMAL(2,1) NOT NULL 
        CHECK (rating >= 0.5 AND rating <= 5.0 AND MOD(rating * 10, 5) = 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint for Upsert semantics
    CONSTRAINT uq_rating_movie_rater UNIQUE (movie_title, rater_id),
    
    -- Foreign key to movies table
    CONSTRAINT fk_ratings_movie 
        FOREIGN KEY (movie_title) 
        REFERENCES movies(title) 
        ON DELETE CASCADE
);

-- Create index for rating aggregation queries
CREATE INDEX IF NOT EXISTS idx_ratings_movie_title ON ratings(movie_title);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_movies_updated_at 
    BEFORE UPDATE ON movies 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ratings_updated_at 
    BEFORE UPDATE ON ratings 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
