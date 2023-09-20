package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
    "os"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	tokenExpirations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitlab_token_expiration_days",
			Help: "Days until GitLab personal access token expiration",
		},
		[]string{"token_name", "token_id", "token_owner"},
	)

	gitlabAPIURL = ""
	gitlabToken = ""
)




type GitLabToken struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	ExpiresAt    string    `json:"expires_at"`
	Username     string    `json:"username"`
	Scopes       []string  `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
	Revoked      bool      `json:"revoked"`
	Active       bool      `json:"active"`
	AccessLevels []string  `json:"access_levels"`
}

func main() {

	fmt.Println("GITLAB_API_URL:", os.Getenv("GITLAB_API_URL"))
	fmt.Println("GITLAB_PERSONAL_ACCESS_TOKEN:", os.Getenv("GITLAB_PERSONAL_ACCESS_TOKEN"))


	// Register the metric collector
	prometheus.MustRegister(tokenExpirations)

	// Set up HTTP server to expose metrics
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	// Periodically check token expiration
	checkTokenExpirations()

	// Run forever
	select {}
}


func checkTokenExpirations() {
	for {
		tokens, err := getAllGitLabTokens()
		if err != nil {
			fmt.Printf("Error fetching GitLab tokens: %v\n", err)
		} else {
			for _, token := range tokens {
				expirationDateStr := token.ExpiresAt

				// If the time part is missing, add a default time (23:59:59)
				if len(expirationDateStr) == 10 {
					expirationDateStr += "T23:59:59Z"
				}

				expirationDate, err := time.Parse(time.RFC3339, expirationDateStr)
				if err != nil {
					fmt.Printf("Error parsing token expiration date: %v\n", err)
					continue
				}

				// Skip setting the metric for tokens with "Private Token" name
				if token.Name == "Private Token" {
					continue
				}

				daysUntilExpiration := int(expirationDate.Sub(time.Now()).Hours() / 24)
				tokenExpirations.WithLabelValues(token.Name, fmt.Sprintf("%d", token.ID), token.Username).Set(float64(daysUntilExpiration))
			}
		}

		time.Sleep(24 * time.Hour) // Check once per day
	}
}


func getAllGitLabTokens() ([]GitLabToken, error) {
	var allTokens []GitLabToken
	page := 1
	for {
		tokens, err := getGitLabTokens(page)
		if err != nil {
			return nil, err
		}

		if len(tokens) == 0 {
			break
		}

		allTokens = append(allTokens, tokens...)
		page++
	}

	return allTokens, nil
}

func getGitLabTokens(page int) ([]GitLabToken, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", gitlabAPIURL+"/personal_access_tokens?page="+fmt.Sprintf("%d", page), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+gitlabToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status code: %d", resp.StatusCode)
	}

	var tokens []GitLabToken
	err = json.NewDecoder(resp.Body).Decode(&tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}