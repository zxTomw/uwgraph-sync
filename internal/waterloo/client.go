package waterloo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxAttempts = 4

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(baseURL, apiKey string, timeout time.Duration, logger *slog.Logger) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

func (c *Client) Terms(ctx context.Context) ([]Term, error) {
	var result []Term
	return result, c.getJSON(ctx, "/v3/Terms", &result)
}

func (c *Client) Subjects(ctx context.Context) ([]Subject, error) {
	var result []Subject
	return result, c.getJSON(ctx, "/v3/Subjects", &result)
}

func (c *Client) AcademicOrganizations(ctx context.Context) ([]AcademicOrganization, error) {
	var result []AcademicOrganization
	return result, c.getJSON(ctx, "/v3/AcademicOrganizations", &result)
}

func (c *Client) Locations(ctx context.Context) ([]Location, error) {
	var result []Location
	return result, c.getJSON(ctx, "/v3/Locations", &result)
}

func (c *Client) Courses(ctx context.Context, termCode string) ([]Course, error) {
	var result []Course
	return result, c.getJSON(ctx, "/v3/Courses/"+url.PathEscape(termCode), &result)
}

func (c *Client) ScheduledCourseIDs(ctx context.Context, termCode string) ([]string, error) {
	var result []string
	return result, c.getJSON(ctx, "/v3/ClassSchedules/"+url.PathEscape(termCode), &result)
}

func (c *Client) Classes(ctx context.Context, termCode, courseID string) ([]Class, error) {
	path := "/v3/ClassSchedules/" + url.PathEscape(termCode) + "/" + url.PathEscape(courseID)
	var result []Class
	return result, c.getJSON(ctx, path, &result)
}

func (c *Client) Exams(ctx context.Context, termCode string) ([]Exam, error) {
	var result []Exam
	return result, c.getJSON(ctx, "/v3/ExamSchedules/"+url.PathEscape(termCode), &result)
}

func (c *Client) getJSON(ctx context.Context, path string, dest any) error {
	endpoint := c.baseURL + path
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-api-key", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if !shouldRetryStatus(0) || attempt == maxAttempts {
				break
			}
			c.sleepBeforeRetry(ctx, path, attempt)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if err := json.Unmarshal(body, dest); err != nil {
				return fmt.Errorf("decode %s: %w", path, err)
			}
			return nil
		}

		lastErr = fmt.Errorf("GET %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
		if !shouldRetryStatus(resp.StatusCode) || attempt == maxAttempts {
			break
		}
		c.sleepBeforeRetry(ctx, path, attempt)
	}

	return lastErr
}

func shouldRetryStatus(status int) bool {
	return status == 0 || status == http.StatusTooManyRequests || status >= 500
}

func (c *Client) sleepBeforeRetry(ctx context.Context, path string, attempt int) {
	delay := time.Duration(math.Pow(2, float64(attempt-1))) * 500 * time.Millisecond
	c.logger.Warn("retry waterloo api request", "path", path, "attempt", attempt+1, "delay", delay)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
