package dictionary

import (
	"context"
	"sync"
)

type LessonCache interface {
	Get(context.Context, string) (*LearningGuide, bool, error)
	Set(context.Context, string, *LearningGuide) error
}

type MemoryLessonCache struct {
	mu      sync.RWMutex
	lessons map[string]*LearningGuide
}

func NewMemoryLessonCache() *MemoryLessonCache {
	return &MemoryLessonCache{lessons: make(map[string]*LearningGuide)}
}

func (c *MemoryLessonCache) Get(_ context.Context, word string) (*LearningGuide, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	lesson, ok := c.lessons[word]
	return lesson, ok, nil
}

func (c *MemoryLessonCache) Set(_ context.Context, word string, lesson *LearningGuide) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lessons[word] = lesson
	return nil
}
