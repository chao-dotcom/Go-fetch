package jobs

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Handler processes a job payload.
type Handler func(ctx context.Context, payload json.RawMessage) (any, error)

// Registry maps job types to handlers.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRegistry creates a registry populated with default demo jobs.
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[string]Handler),
	}
	r.Register("send_email", ExecuteEmailJob)
	r.Register("process_video", ExecuteVideoJob)
	r.Register("generate_thumbnail", ExecuteThumbnailJob)
	r.Register("webhook", ExecuteWebhookJob)
	return r
}

// Register adds a handler.
func (r *Registry) Register(name string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = handler
}

// HandlerFor retrieves the handler for a job type.
func (r *Registry) HandlerFor(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[name]
	return h, ok
}

// ListTypes returns the registered job types.
func (r *Registry) ListTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.handlers))
	for key := range r.handlers {
		types = append(types, key)
	}
	return types
}

// ----------------------------------------------------------------------
// Email Job
// ----------------------------------------------------------------------

type EmailJobPayload struct {
	To      string            `json:"to"`
	Subject string            `json:"subject"`
	Body    string            `json:"body"`
	From    string            `json:"from,omitempty"`
	CC      []string          `json:"cc,omitempty"`
	BCC     []string          `json:"bcc,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type EmailJobResult struct {
	MessageID  string    `json:"message_id"`
	SentAt     time.Time `json:"sent_at"`
	Recipients int       `json:"recipients"`
}

// ExecuteEmailJob simulates sending an email via SMTP.
func ExecuteEmailJob(ctx context.Context, payload json.RawMessage) (any, error) {
	var params EmailJobPayload
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid email payload: %w", err)
	}
	if params.To == "" {
		return nil, errors.New("recipient email is required")
	}
	if params.Subject == "" {
		return nil, errors.New("subject is required")
	}
	if params.From == "" {
		params.From = "noreply@taskqueue.local"
	}

	message := buildEmailMessage(params)
	if err := sendEmail(ctx, params.From, []string{params.To}, message); err != nil {
		return nil, err
	}

	return EmailJobResult{
		MessageID:  generateMessageID(),
		SentAt:     time.Now(),
		Recipients: 1 + len(params.CC) + len(params.BCC),
	}, nil
}

func buildEmailMessage(params EmailJobPayload) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s\r\n", params.From))
	b.WriteString(fmt.Sprintf("To: %s\r\n", params.To))
	if len(params.CC) > 0 {
		b.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(params.CC, ", ")))
	}
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", params.Subject))
	for k, v := range params.Headers {
		b.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(params.Body)
	return b.String()
}

// sendEmail mocks an SMTP delivery with context awareness.
func sendEmail(ctx context.Context, from string, recipients []string, message string) error {
	_ = from
	_ = recipients
	_ = message

	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func generateMessageID() string {
	return fmt.Sprintf("<%s@taskqueue.local>", generateID())
}

// ----------------------------------------------------------------------
// Video Job
// ----------------------------------------------------------------------

type VideoJobPayload struct {
	URL        string `json:"url"`
	Format     string `json:"format"`
	Resolution string `json:"resolution"`
	Codec      string `json:"codec,omitempty"`
	Bitrate    string `json:"bitrate,omitempty"`
}

type VideoJobResult struct {
	OutputURL      string  `json:"output_url"`
	Duration       float64 `json:"duration_seconds"`
	FileSize       int64   `json:"file_size_bytes"`
	ProcessingTime float64 `json:"processing_time_seconds"`
}

func ExecuteVideoJob(ctx context.Context, payload json.RawMessage) (any, error) {
	var params VideoJobPayload
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid video payload: %w", err)
	}
	if params.URL == "" {
		return nil, errors.New("video URL is required")
	}
	if params.Format == "" {
		params.Format = "mp4"
	}
	if params.Resolution == "" {
		params.Resolution = "1080p"
	}

	start := time.Now()
	if err := downloadVideo(ctx, params.URL); err != nil {
		return nil, err
	}
	outputURL, fileSize, duration, err := processVideo(ctx, params)
	if err != nil {
		return nil, err
	}

	return VideoJobResult{
		OutputURL:      outputURL,
		Duration:       duration,
		FileSize:       fileSize,
		ProcessingTime: time.Since(start).Seconds(),
	}, nil
}

func downloadVideo(ctx context.Context, url string) error {
	_ = url
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return nil
}

func processVideo(ctx context.Context, params VideoJobPayload) (string, int64, float64, error) {
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return "", 0, 0, ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}

	return fmt.Sprintf("https://cdn.taskqueue.local/video/%s_%s.%s", generateID(), params.Resolution, params.Format),
		15_000_000,
		120.5,
		nil
}

// ----------------------------------------------------------------------
// Thumbnail Job
// ----------------------------------------------------------------------

type ThumbnailJobPayload struct {
	VideoID   string  `json:"video_id"`
	Timestamp float64 `json:"timestamp"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
}

type ThumbnailJobResult struct {
	ThumbnailURL string `json:"thumbnail_url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int64  `json:"file_size_bytes"`
}

func ExecuteThumbnailJob(ctx context.Context, payload json.RawMessage) (any, error) {
	var params ThumbnailJobPayload
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid thumbnail payload: %w", err)
	}
	if params.VideoID == "" {
		return nil, errors.New("video_id is required")
	}
	if params.Width == 0 {
		params.Width = 1280
	}
	if params.Height == 0 {
		params.Height = 720
	}

	timer := time.NewTimer(250 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}

	return ThumbnailJobResult{
		ThumbnailURL: fmt.Sprintf("https://cdn.taskqueue.local/thumb/%s_%d.jpg", params.VideoID, int(params.Timestamp)),
		Width:        params.Width,
		Height:       params.Height,
		FileSize:     125_000,
	}, nil
}

// ----------------------------------------------------------------------
// Webhook Job
// ----------------------------------------------------------------------

type WebhookJobPayload struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Payload interface{}       `json:"payload"`
	Timeout int               `json:"timeout"`
}

type WebhookJobResult struct {
	StatusCode   int               `json:"status_code"`
	ResponseBody string            `json:"response_body"`
	Headers      map[string]string `json:"headers"`
	DurationMS   float64           `json:"duration_ms"`
}

func ExecuteWebhookJob(ctx context.Context, payload json.RawMessage) (any, error) {
	var params WebhookJobPayload
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid webhook payload: %w", err)
	}
	if params.URL == "" {
		return nil, errors.New("url is required")
	}
	if params.Method == "" {
		params.Method = http.MethodPost
	}
	if params.Timeout == 0 {
		params.Timeout = 30
	}

	bodyBytes := []byte{}
	if params.Payload != nil {
		var err error
		bodyBytes, err = json.Marshal(params.Payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, params.Method, params.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range params.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: time.Duration(params.Timeout) * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}

	result := WebhookJobResult{
		StatusCode:   resp.StatusCode,
		ResponseBody: string(body),
		Headers:      headers,
		DurationMS:   time.Since(start).Seconds() * 1000,
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return result, nil
}

// ----------------------------------------------------------------------
// Utilities
// ----------------------------------------------------------------------

func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

