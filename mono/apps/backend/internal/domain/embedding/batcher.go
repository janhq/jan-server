package embedding

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Batcher handles batch processing of embedding requests
type Batcher struct {
	client    Client
	batchSize int
	timeout   time.Duration

	mu       sync.Mutex
	queue    []batchItem
	timer    *time.Timer
	stopCh   chan struct{}
	resultCh chan batchResult
}

type batchItem struct {
	text     string
	resultCh chan<- batchResult
}

type batchResult struct {
	embedding []float32
	err       error
}

// NewBatcher creates a new embedding batcher
func NewBatcher(client Client, batchSize int, timeout time.Duration) *Batcher {
	b := &Batcher{
		client:    client,
		batchSize: batchSize,
		timeout:   timeout,
		queue:     make([]batchItem, 0, batchSize),
		stopCh:    make(chan struct{}),
		resultCh:  make(chan batchResult, batchSize),
	}

	go b.run()
	return b
}

// Embed adds a text to the batch queue and returns the embedding
func (b *Batcher) Embed(ctx context.Context, text string) ([]float32, error) {
	resultCh := make(chan batchResult, 1)

	b.mu.Lock()
	b.queue = append(b.queue, batchItem{
		text:     text,
		resultCh: resultCh,
	})

	// Start timer if this is the first item
	if len(b.queue) == 1 {
		b.timer = time.AfterFunc(b.timeout, func() {
			b.flush()
		})
	}

	// Flush if batch is full
	if len(b.queue) >= b.batchSize {
		if b.timer != nil {
			b.timer.Stop()
		}
		b.mu.Unlock()
		b.flush()
	} else {
		b.mu.Unlock()
	}

	// Wait for result
	select {
	case result := <-resultCh:
		return result.embedding, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// flush processes the current batch
func (b *Batcher) flush() {
	b.mu.Lock()
	if len(b.queue) == 0 {
		b.mu.Unlock()
		return
	}

	items := b.queue
	b.queue = make([]batchItem, 0, b.batchSize)
	b.mu.Unlock()

	// Extract texts
	texts := make([]string, len(items))
	for i, item := range items {
		texts[i] = item.text
	}

	log.Debug().
		Int("batch_size", len(texts)).
		Msg("Processing embedding batch")

	// Batch embed
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	embeddings, err := b.client.Embed(ctx, texts)

	// Send results back
	for i, item := range items {
		result := batchResult{err: err}
		if err == nil && i < len(embeddings) {
			result.embedding = embeddings[i]
		}

		select {
		case item.resultCh <- result:
		default:
			log.Warn().Msg("Failed to send batch result")
		}
	}
}

// run is the background goroutine that handles batch processing
func (b *Batcher) run() {
	ticker := time.NewTicker(b.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-b.stopCh:
			b.flush()
			return
		}
	}
}

// Stop stops the batcher
func (b *Batcher) Stop() {
	close(b.stopCh)
}
