package dictionary

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type WordSource interface {
	Lookup(context.Context, string) (WordDetails, error)
}

type LessonGenerator interface {
	Generate(context.Context, WordDetails) (*LearningGuide, error)
}

type LookupResult struct {
	Details       WordDetails
	LessonWarning error
}

type Service struct {
	source   WordSource
	teacher  LessonGenerator
	cache    LessonCache
	requests singleflight.Group
	sources  singleflight.Group
	mu       sync.Mutex
	entries  map[string]cachedEntry
}

func NewService(source WordSource, teacher LessonGenerator, cache LessonCache) *Service {
	return &Service{source: source, teacher: teacher, cache: cache, entries: make(map[string]cachedEntry)}
}

type cachedEntry struct {
	details WordDetails
	expires time.Time
}

// Preview never invokes the teaching model. Valid saved lessons can answer even
// while the external dictionary is unavailable.
func (s *Service) Preview(ctx context.Context, rawWord string) (LookupResult, error) {
	word, err := NormalizeWord(rawWord)
	if err != nil {
		return LookupResult{}, err
	}
	if cached, ok := s.cachedLesson(ctx, word); ok {
		return cached, nil
	}
	details, err := s.entry(ctx, word)
	return LookupResult{Details: details}, err
}

func (s *Service) Entry(ctx context.Context, rawWord string) (LookupResult, error) {
	word, err := NormalizeWord(rawWord)
	if err != nil {
		return LookupResult{}, err
	}
	details, err := s.entry(ctx, word)
	return LookupResult{Details: details}, err
}

func (s *Service) cachedLesson(ctx context.Context, word string) (LookupResult, bool) {
	if s.cache == nil {
		return LookupResult{}, false
	}
	lesson, ok, err := s.cache.Get(ctx, word)
	if !ok || lesson == nil || lesson.Validate() != nil {
		return LookupResult{}, false
	}
	s.mu.Lock()
	entry := s.entries[word]
	s.mu.Unlock()
	details := entry.details
	details.Word = word
	details.SourceURL = "https://en.wiktionary.org/wiki/" + url.PathEscape(word)
	if details.Entries == nil {
		details.Entries = []DictionaryEntry{}
	}
	details.Learning = lesson
	return LookupResult{Details: details, LessonWarning: err}, true
}

func (s *Service) entry(ctx context.Context, word string) (WordDetails, error) {
	s.mu.Lock()
	cached, ok := s.entries[word]
	s.mu.Unlock()
	if ok && time.Now().Before(cached.expires) {
		return cached.details, nil
	}
	result := s.sources.DoChan(word, func() (any, error) {
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 12*time.Second)
		defer cancel()
		details, err := s.source.Lookup(fetchCtx, word)
		if err == nil {
			s.mu.Lock()
			if len(s.entries) >= 512 {
				clear(s.entries)
			}
			s.entries[word] = cachedEntry{details: details, expires: time.Now().Add(time.Hour)}
			s.mu.Unlock()
		}
		return details, err
	})
	select {
	case <-ctx.Done():
		return WordDetails{}, ctx.Err()
	case out := <-result:
		if out.Err != nil {
			return WordDetails{}, out.Err
		}
		return out.Val.(WordDetails), nil
	}
}

func (s *Service) Lookup(ctx context.Context, rawWord string) (LookupResult, error) {
	word, err := NormalizeWord(rawWord)
	if err != nil {
		return LookupResult{}, err
	}
	if cached, ok := s.cachedLesson(ctx, word); ok {
		return cached, nil
	}
	result := s.requests.DoChan(word, func() (any, error) {
		workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 75*time.Second)
		defer cancel()
		if cached, ok := s.cachedLesson(workCtx, word); ok {
			return cached, nil
		}
		details, err := s.entry(workCtx, word)
		if err != nil {
			return LookupResult{}, err
		}
		if s.teacher == nil {
			return LookupResult{Details: details}, nil
		}
		lesson, err := s.teacher.Generate(workCtx, details)
		if err == nil && lesson == nil {
			err = errors.New("learning agent returned no guide")
		}
		if err == nil {
			err = lesson.Validate()
		}
		if err != nil {
			details.LearningError = lessonErrorMessage(err)
			return LookupResult{Details: details, LessonWarning: err}, nil
		}
		details.Learning = lesson
		if s.cache != nil {
			err = s.cache.Set(workCtx, word, lesson)
		}
		return LookupResult{Details: details, LessonWarning: err}, nil
	})
	select {
	case <-ctx.Done():
		return LookupResult{}, ctx.Err()
	case out := <-result:
		if out.Err != nil {
			return LookupResult{}, out.Err
		}
		return out.Val.(LookupResult), nil
	}
}
