package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"src/internal/biz"
	"src/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

type boxOfficeClient struct {
	client     *http.Client
	baseURL    string
	apiKey     string
	maxRetries int
	log        *log.Helper
}

// NewBoxOfficeClient creates a new box office API client
func NewBoxOfficeClient(c *conf.BoxOffice, logger log.Logger) biz.BoxOfficeClient {
	return &boxOfficeClient{
		client: &http.Client{
			Timeout: c.Timeout.AsDuration(),
		},
		baseURL:    c.Url,
		apiKey:     c.ApiKey,
		maxRetries: int(c.MaxRetries),
		log:        log.NewHelper(logger),
	}
}

func (c *boxOfficeClient) GetBoxOffice(ctx context.Context, title string) (*biz.BoxOfficeData, error) {
	var lastErr error

	// Retry logic with exponential backoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * 100 * time.Millisecond
			time.Sleep(backoff)
			c.log.Infof("retrying box office request for '%s', attempt %d/%d", title, attempt, c.maxRetries)
		}

		data, err := c.doRequest(ctx, title)
		if err == nil {
			return data, nil
		}

		lastErr = err

		// Don't retry on 404
		if err.Error() == "not found" {
			break
		}
	}

	// Return nil on failure (non-blocking)
	c.log.Warnf("box office request failed after %d attempts: %v", c.maxRetries+1, lastErr)
	return nil, lastErr
}

func (c *boxOfficeClient) doRequest(ctx context.Context, title string) (*biz.BoxOfficeData, error) {
	url := fmt.Sprintf("%s/boxoffice?title=%s", c.baseURL, title)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		Title       string `json:"title"`
		Distributor string `json:"distributor"`
		ReleaseDate string `json:"releaseDate"`
		Budget      int64  `json:"budget"`
		Revenue     struct {
			Worldwide         int64 `json:"worldwide"`
			OpeningWeekendUSA int64 `json:"openingWeekendUSA"`
		} `json:"revenue"`
		MPARating string `json:"mpaRating"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to biz model
	data := &biz.BoxOfficeData{
		Title:       response.Title,
		Distributor: &response.Distributor,
		Budget:      &response.Budget,
		MPARating:   &response.MPARating,
		Revenue: &biz.BoxOfficeRevenue{
			Worldwide:         response.Revenue.Worldwide,
			OpeningWeekendUSA: &response.Revenue.OpeningWeekendUSA,
		},
	}

	// Parse release date if provided
	if response.ReleaseDate != "" {
		if t, err := time.Parse("2006-01-02", response.ReleaseDate); err == nil {
			data.ReleaseDate = &t
		}
	}

	return data, nil
}
