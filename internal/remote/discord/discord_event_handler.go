package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/event"
)

func (b *Bot) Handle(ctx context.Context, e event.Event) error {
	if b.shouldPublish(e) {
		return b.sendMessageWithImage(e)
	}
	return b.sendSimpleMessage(e.Message())
}

func (b *Bot) sendMessageWithImage(e event.Event) error {
	// Create a buffer for the JPEG image
	imgBuf := new(bytes.Buffer)
	err := jpeg.Encode(imgBuf, e.Image(), &jpeg.Options{Quality: 80})
	if err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	// Create a buffer for the multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the message content as a field
	payload := map[string]string{
		"content": e.Message(),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	err = writer.WriteField("payload_json", string(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to write payload field: %w", err)
	}

	// Add the image file
	part, err := writer.CreateFormFile("file", "Screenshot.jpeg")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, imgBuf)
	if err != nil {
		return fmt.Errorf("failed to copy image to form: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create and send the request
	req, err := http.NewRequest("POST", b.webhookURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("webhook returned status code %d", resp.StatusCode)
	}

	return nil
}

func (b *Bot) sendSimpleMessage(content string) error {
	return b.handleWebhookMessage(content)
}

func (b *Bot) shouldPublish(e event.Event) bool {
	if e.Image() == nil {
		return false
	}
	switch evt := e.(type) {
	case event.GameFinishedEvent:
		if evt.Reason == event.FinishedChicken && !config.Koolo.Discord.EnableDiscordChickenMessages {
			return false
		}
		if evt.Reason == event.FinishedOK && !config.Koolo.Discord.EnableRunFinishMessages {
			return false
		}
		if evt.Reason == event.FinishedError && !config.Koolo.Discord.EnableGameCreatedMessages {
			return false
		}
	case event.GameCreatedEvent:
		if !config.Koolo.Discord.EnableGameCreatedMessages {
			return false
		}
	}
	return true
}
