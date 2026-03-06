package slack

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

// SearchMessagesInput defines input for searching messages
type SearchMessagesInput struct {
	Query string `json:"query" jsonschema:"Search query (supports Slack search modifiers like from:@user, in:#channel, before:date)"`
	Count int    `json:"count,omitempty" jsonschema:"Number of results to return (default 20, max 100)"`
	Sort  string `json:"sort,omitempty" jsonschema:"Sort order: score (relevance) or timestamp (recent first)"`
}

// SearchMatch represents a search result
type SearchMatch struct {
	Timestamp string `json:"timestamp"`
	Channel   string `json:"channel"`
	User      string `json:"user"`
	UserName  string `json:"user_name,omitempty"`
	Text      string `json:"text"`
	Permalink string `json:"permalink"`
}

// SearchMessagesOutput contains search results
type SearchMessagesOutput struct {
	Query   string        `json:"query"`
	Total   int           `json:"total"`
	Matches []SearchMatch `json:"matches"`
}

// SearchMessages searches messages across the workspace
func (c *Service) SearchMessages(ctx context.Context, input SearchMessagesInput) (SearchMessagesOutput, error) {
	count := 20
	if input.Count > 0 && input.Count <= 100 {
		count = input.Count
	}

	sort := "score"
	if input.Sort == "timestamp" {
		sort = "timestamp"
	}

	params := slack.SearchParameters{
		Sort:          sort,
		SortDirection: "desc",
		Count:         count,
	}

	results, err := c.searchMessages(ctx, input.Query, params)
	if err != nil {
		return SearchMessagesOutput{}, fmt.Errorf("failed to search: %w", err)
	}

	output := SearchMessagesOutput{
		Query:   input.Query,
		Total:   results.Total,
		Matches: make([]SearchMatch, 0, len(results.Matches)),
	}

	for _, match := range results.Matches {
		output.Matches = append(output.Matches, SearchMatch{
			Timestamp: match.Timestamp,
			Channel:   match.Channel.Name,
			User:      match.User,
			UserName:  match.Username,
			Text:      match.Text,
			Permalink: match.Permalink,
		})
	}

	return output, nil
}
