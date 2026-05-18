package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

type Comics struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type SearchReply struct {
	Comics []Comics `json:"comics"`
	Total  int      `json:"total"`
}

type UpdateStats struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

type UpdateStatus struct {
	Status string `json:"status"`
}

type apiError struct {
	StatusCode int
	Message    string
}

func (e apiError) Error() string {
	return fmt.Sprintf("api response %d: %s", e.StatusCode, e.Message)
}

func NewAPIClient(baseURL string, timeout time.Duration) *APIClient {
	base := strings.TrimRight(baseURL, "/")
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &APIClient{
		baseURL: base,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *APIClient) Login(ctx context.Context, name, password string) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"name":     name,
		"password": password,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/login",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", apiError{
			StatusCode: resp.StatusCode,
			Message:    strings.TrimSpace(string(body)),
		}
	}

	token, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(token), nil
}

func (c *APIClient) Search(ctx context.Context, phrase string, limit int, useIndex bool) (SearchReply, error) {
	values := url.Values{}
	values.Set("phrase", phrase)
	if limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", limit))
	}

	path := "/api/search"
	if useIndex {
		path = "/api/isearch"
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+path+"?"+values.Encode(),
		nil,
	)
	if err != nil {
		return SearchReply{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchReply{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return SearchReply{}, apiError{
			StatusCode: resp.StatusCode,
			Message:    strings.TrimSpace(string(body)),
		}
	}

	var reply SearchReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return SearchReply{}, err
	}

	return reply, nil
}

func (c *APIClient) Update(ctx context.Context, token string) (int, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/db/update",
		nil,
	)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Token "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func (c *APIClient) Drop(ctx context.Context, token string) (int, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		c.baseURL+"/api/db",
		nil,
	)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Token "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func (c *APIClient) Stats(ctx context.Context) (UpdateStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/db/stats", nil)
	if err != nil {
		return UpdateStats{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return UpdateStats{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return UpdateStats{}, apiError{
			StatusCode: resp.StatusCode,
			Message:    strings.TrimSpace(string(body)),
		}
	}

	var stats UpdateStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return UpdateStats{}, err
	}

	return stats, nil
}

func (c *APIClient) Status(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/db/status", nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", apiError{
			StatusCode: resp.StatusCode,
			Message:    strings.TrimSpace(string(body)),
		}
	}

	var status UpdateStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return "", err
	}

	return status.Status, nil
}
