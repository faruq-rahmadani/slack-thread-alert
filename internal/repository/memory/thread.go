package memory

import (
	"context"
	"sync"
	"time"

	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
)

const (
	// DefaultThreadTTL keeps a thread for alerts that never recover (e.g. log alerts)
	DefaultThreadTTL = 7 * 24 * time.Hour

	// DefaultEventTTL covers Datadog retries (15s timeout x 5 attempts) with margin
	DefaultEventTTL = time.Hour
)

// ThreadRepository is an in-memory implementation of usecase.ThreadRepositoryItf.
// Data is lost on restart and not shared between instances.
type ThreadRepository struct {
	mu        sync.Mutex
	threads   map[string]*domain.Thread
	events    map[string]time.Time // eventID -> processed at
	threadTTL time.Duration
	eventTTL  time.Duration
	now       func() time.Time
}

func NewThreadRepository(threadTTL, eventTTL time.Duration) *ThreadRepository {
	return &ThreadRepository{
		threads:   make(map[string]*domain.Thread),
		events:    make(map[string]time.Time),
		threadTTL: threadTTL,
		eventTTL:  eventTTL,
		now:       time.Now,
	}
}

func (r *ThreadRepository) GetThread(_ context.Context, key string) (*domain.Thread, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[key]
	if !ok || r.isExpired(thread.CreatedAt, r.threadTTL) {
		delete(r.threads, key)
		return nil, domain.ErrThreadNotFound
	}

	copied := *thread
	return &copied, nil
}

func (r *ThreadRepository) SaveThread(_ context.Context, thread *domain.Thread) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pruneThreads()

	copied := *thread
	if copied.CreatedAt.IsZero() {
		copied.CreatedAt = r.now()
	}
	r.threads[copied.Key] = &copied

	return nil
}

func (r *ThreadRepository) DeleteThread(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.threads, key)

	return nil
}

func (r *ThreadRepository) MarkEventProcessed(_ context.Context, eventID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pruneEvents()

	if _, ok := r.events[eventID]; ok {
		return false, nil
	}
	r.events[eventID] = r.now()

	return true, nil
}

func (r *ThreadRepository) UnmarkEventProcessed(_ context.Context, eventID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.events, eventID)

	return nil
}

func (r *ThreadRepository) isExpired(createdAt time.Time, ttl time.Duration) bool {
	return r.now().Sub(createdAt) > ttl
}

// pruneThreads removes expired threads, caller must hold the lock
func (r *ThreadRepository) pruneThreads() {
	for key, thread := range r.threads {
		if r.isExpired(thread.CreatedAt, r.threadTTL) {
			delete(r.threads, key)
		}
	}
}

// pruneEvents removes expired events, caller must hold the lock
func (r *ThreadRepository) pruneEvents() {
	for id, processedAt := range r.events {
		if r.isExpired(processedAt, r.eventTTL) {
			delete(r.events, id)
		}
	}
}
