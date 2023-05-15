package emoji

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type SlackEmoji struct {
	Name            string   `json:"name"`
	IsAlias         int      `json:"is_alias"`
	AliasFor        string   `json:"alias_for"`
	URL             string   `json:"url"`
	TeamID          string   `json:"team_id"`
	UserID          string   `json:"user_id"`
	Created         int      `json:"created"`
	IsBad           bool     `json:"is_bad"`
	UserDisplayName string   `json:"user_display_name"`
	AvatarHas       string   `json:"avatar_hash"`
	CanDelete       bool     `json:"can_delete"`
	Synonyms        []string `json:"synonyms"`
}

type SlackExporter struct {
	config Config

	client http.Client

	store *Store

	logger *log.Logger
}

func NewSlackExporter(store *Store, config Config) SlackExporter {
	return SlackExporter{
		config: config,
		store:  store,
		logger: log.Default(),
	}
}

func (e SlackExporter) Run(ctx context.Context) error {
	emojiC := make(chan SlackEmoji)

	total, err := e.checkEmojiCount(ctx)
	if err != nil {
		return fmt.Errorf("check emoji count: %w", err)
	}

	// List metadata about emojis
	errC := make(chan error)
	go func() {
		if err := e.listEmoji(ctx, total, emojiC); err != nil {
			errC <- fmt.Errorf("list emoji: %w", err)
		}
	}()

	// Fetch emoji image files and store them on disk
	for i := 0; i < total; i += 1 {
		var emoji SlackEmoji
		select {
		case err := <-errC:
			return fmt.Errorf("list request: %w", err)
		case <-ctx.Done():
			return ctx.Err()
		case emoji = <-emojiC:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", emoji.URL, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		res, err := e.client.Do(req)
		if err != nil {
			return fmt.Errorf("do request: %w", err)
		}
		defer res.Body.Close()

		e.logger.Printf("Storing %q", emoji.Name)
		if err := e.store.Store(emoji, res.Body); err != nil {
			return fmt.Errorf("store emoji: %w", err)
		}
	}

	close(emojiC)

	return nil
}

func (e SlackExporter) checkEmojiCount(ctx context.Context) (int, error) {
	page := 1
	size := 1
	res, err := e.getEmojiPage(ctx, page, size)
	if err != nil {
		return 0, fmt.Errorf("first list request: %w", err)
	}

	return res.Paging.Total, nil
}

func (e SlackExporter) listEmoji(ctx context.Context, total int, emojiC chan<- SlackEmoji) error {
	const pageSize = 100

	for page := 1; page < total/pageSize; page += 1 {
		e.logger.Printf("Listing emoji page #%d", page)

		list, err := e.getEmojiPage(ctx, page, pageSize)
		if err != nil {
			return fmt.Errorf("list request for page #%d: %w", page, err)
		}

		for _, emoji := range list.Emoji {
			emojiC <- emoji
		}
	}

	return nil
}

func (e SlackExporter) getEmojiPage(ctx context.Context, page int, size int) (*ListResponse, error) {
	form := make(url.Values)
	form.Add("token", e.config.Slack.Token)
	form.Add("page", strconv.FormatInt(int64(page), 10))

	u, _ := url.Parse("https://dd.slack.com/api/emoji.adminList")
	u.Query().Add("slack_route", e.config.Slack.Route)

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header = make(http.Header)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var list ListResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &list, nil
}

type ListResponse struct {
	OK     bool         `json:"ok"`
	Emoji  []SlackEmoji `json:"emoji"`
	Paging struct {
		Count int `json:"count"`
		Total int `json:"total"`
		Page  int `json:"page"`
		Pages int `json:"pages"`
	} `json:"paging"`
}
