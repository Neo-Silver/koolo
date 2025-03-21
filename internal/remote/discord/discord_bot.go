package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/hectorgimenez/koolo/internal/bot"
	"github.com/hectorgimenez/koolo/internal/config"
)

type Bot struct {
	webhookURL string // Using webhook URL directly
	manager    *bot.SupervisorManager
}

func NewBot(webhookURL, channelID string, manager *bot.SupervisorManager) (*Bot, error) {
	// In this case, we are skipping discordgo.Session and simply storing the webhook URL
	return &Bot{
		webhookURL: webhookURL,
		manager:    manager,
	}, nil
}

func (b *Bot) Start(ctx context.Context) error {
	// Simulate listening for messages (as if they were from Discord session events)
	<-ctx.Done()
	return nil // No actual session to close in this setup
}

func (b *Bot) handleWebhookMessage(content string) error {
	// Send a message via webhook
	message := map[string]string{"content": content}
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	resp, err := http.Post(b.webhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error sending message to webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("webhook returned status code %d", resp.StatusCode)
	}

	return nil
}

func (b *Bot) onMessageCreated(content, authorID string) {
	// Check if the message is from a bot admin
	if !slices.Contains(config.Koolo.Discord.BotAdmins, authorID) {
		return
	}

	prefix := strings.Split(content, " ")[0]
	switch prefix {
	case "!start":
		b.handleStartRequest(content)
	case "!stop":
		b.handleStopRequest(content)
	case "!stats":
		b.handleStatsRequest(content)
	case "!status":
		b.handleStatusRequest(content)
	}
}

func (b *Bot) handleStartRequest(content string) {
	words := strings.Fields(content)

	if len(words) > 1 {
		for _, supervisor := range words[1:] {
			if !b.supervisorExists(supervisor) {
				b.handleWebhookMessage(fmt.Sprintf("Supervisor '%s' not found.", supervisor))
				continue
			}

			b.manager.Start(supervisor, false)
			time.Sleep(1 * time.Second)
			b.handleWebhookMessage(fmt.Sprintf("Supervisor '%s' has been started.", supervisor))
		}
	} else {
		b.handleWebhookMessage("Usage: !start <supervisor1> [supervisor2] ...")
	}
}

func (b *Bot) handleStopRequest(content string) {
	words := strings.Fields(content)

	if len(words) > 1 {
		for _, supervisor := range words[1:] {
			if !b.supervisorExists(supervisor) {
				b.handleWebhookMessage(fmt.Sprintf("Supervisor '%s' not found.", supervisor))
				continue
			}

			if b.manager.Status(supervisor).SupervisorStatus == bot.NotStarted || b.manager.Status(supervisor).SupervisorStatus == "" {
				b.handleWebhookMessage(fmt.Sprintf("Supervisor '%s' is not running.", supervisor))
				continue
			}

			b.manager.Stop(supervisor)
			time.Sleep(1 * time.Second)
			b.handleWebhookMessage(fmt.Sprintf("Supervisor '%s' has been stopped.", supervisor))
		}
	} else {
		b.handleWebhookMessage("Usage: !stop <supervisor1> [supervisor2] ...")
	}
}

func (b *Bot) handleStatusRequest(content string) {
	words := strings.Fields(content)

	if len(words) > 1 {
		for _, supervisor := range words[1:] {
			if !b.supervisorExists(supervisor) {
				b.handleWebhookMessage(fmt.Sprintf("Supervisor '%s' not found.", supervisor))
				continue
			}

			status := b.manager.Status(supervisor)
			if status.SupervisorStatus == bot.NotStarted || status.SupervisorStatus == "" {
				b.handleWebhookMessage(fmt.Sprintf("Supervisor '%s' is offline.", supervisor))
				continue
			}

			b.handleWebhookMessage(fmt.Sprintf("Supervisor '%s' is %s", supervisor, status.SupervisorStatus))
		}
	} else {
		b.handleWebhookMessage("Usage: !status <supervisor1> [supervisor2] ...")
	}
}

func (b *Bot) handleStatsRequest(content string) {
	words := strings.Fields(content)

	if len(words) > 1 {
		for _, supervisor := range words[1:] {
			if !b.supervisorExists(supervisor) {
				b.handleWebhookMessage(fmt.Sprintf("Supervisor '%s' not found.", supervisor))
				continue
			}

			supStatus := string(b.manager.Status(supervisor).SupervisorStatus)
			if supStatus == string(bot.NotStarted) || supStatus == "" {
				supStatus = "Offline"
			}

			// Create webhook embed message
			embedMsg := map[string]interface{}{
				"embeds": []map[string]interface{}{
					{
						"title": fmt.Sprintf("Stats for %s", supervisor),
						"fields": []map[string]interface{}{
							{
								"name":   "Status",
								"value":  supStatus,
								"inline": true,
							},
							{
								"name":   "Uptime",
								"value":  time.Since(b.manager.Status(supervisor).StartedAt).String(),
								"inline": true,
							},
							{
								"name":   "Games",
								"value":  fmt.Sprintf("%d", b.manager.GetSupervisorStats(supervisor).TotalGames()),
								"inline": true,
							},
							{
								"name":   "Drops",
								"value":  fmt.Sprintf("%d", len(b.manager.GetSupervisorStats(supervisor).Drops)),
								"inline": true,
							},
							{
								"name":   "Deaths",
								"value":  fmt.Sprintf("%d", b.manager.GetSupervisorStats(supervisor).TotalDeaths()),
								"inline": true,
							},
							{
								"name":   "Chickens",
								"value":  fmt.Sprintf("%d", b.manager.GetSupervisorStats(supervisor).TotalChickens()),
								"inline": true,
							},
							{
								"name":   "Errors",
								"value":  fmt.Sprintf("%d", b.manager.GetSupervisorStats(supervisor).TotalErrors()),
								"inline": true,
							},
						},
					},
				},
			}

			body, err := json.Marshal(embedMsg)
			if err != nil {
				b.handleWebhookMessage(fmt.Sprintf("Error creating stats message for '%s': %v", supervisor, err))
				continue
			}

			// Send the embed message using the webhook
			if err := b.sendWebhookJSON(body); err != nil {
				b.handleWebhookMessage(fmt.Sprintf("Error sending stats for '%s': %v", supervisor, err))
			}
		}
	} else {
		b.handleWebhookMessage("Usage: !stats <supervisor1> [supervisor2] ...")
	}
}

func (b *Bot) sendWebhookJSON(jsonBody []byte) error {
	resp, err := http.Post(b.webhookURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error sending message to webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("webhook returned status code %d", resp.StatusCode)
	}

	return nil
}
