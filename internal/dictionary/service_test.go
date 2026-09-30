package dictionary

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type fakeWordSource struct {
	details WordDetails
	word    string
}

type sourceFunc func(context.Context, string) (WordDetails, error)

func (f sourceFunc) Lookup(ctx context.Context, word string) (WordDetails, error) {
	return f(ctx, word)
}

type teacherFunc func(context.Context, WordDetails) (*LearningGuide, error)

func (f teacherFunc) Generate(ctx context.Context, details WordDetails) (*LearningGuide, error) {
	return f(ctx, details)
}

func validGuide() *LearningGuide {
	return &LearningGuide{Meaning: "house", Summary: "An everyday noun", GrammarNote: "Feminine: la casa, le case", Examples: []LearningExample{
		{Italian: "Sono a casa.", English: "I am at home."}, {Italian: "La casa e grande.", English: "The house is big."},
		{Italian: "Vado a casa.", English: "I am going home."}, {Italian: "Questa casa e nuova.", English: "This house is new."},
	}}
}

func TestCachedLessonSkipsExternalServices(t *testing.T) {
	cache := NewMemoryLessonCache()
	_ = cache.Set(context.Background(), "casa", validGuide())
	source := sourceFunc(func(context.Context, string) (WordDetails, error) {
		t.Error("cached lesson fetched dictionary")
		return WordDetails{}, errors.New("offline")
	})
	teacher := teacherFunc(func(context.Context, WordDetails) (*LearningGuide, error) {
		t.Error("cached lesson invoked teacher")
		return nil, errors.New("offline")
	})
	s := NewService(source, teacher, cache)
	for _, lookup := range []func(context.Context, string) (LookupResult, error){s.Lookup, s.Preview} {
		result, err := lookup(context.Background(), "CASA")
		if err != nil || result.Details.Learning == nil || result.Details.Learning.Meaning != "house" {
			t.Fatalf("cache miss: %+v %v", result, err)
		}
	}
}

func TestPreviewDoesNotWaitForTeaching(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	var calls atomic.Int32
	s := NewService(sourceFunc(func(context.Context, string) (WordDetails, error) {
		calls.Add(1)
		return WordDetails{Word: "casa"}, nil
	}),
		teacherFunc(func(ctx context.Context, _ WordDetails) (*LearningGuide, error) {
			close(started)
			select {
			case <-release:
				return validGuide(), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}), NewMemoryLessonCache())
	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _, err := s.Lookup(ctx, "casa"); done <- err }()
	<-started
	previewCtx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	result, err := s.Preview(previewCtx, "casa")
	if err != nil || result.Details.Word != "casa" || result.Details.Learning != nil {
		t.Fatalf("preview blocked: %+v %v", result, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("dictionary calls=%d", calls.Load())
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("caller did not cancel: %v", err)
	}
}

func TestDictionaryFailureIsRetried(t *testing.T) {
	var calls int
	s := NewService(sourceFunc(func(context.Context, string) (WordDetails, error) {
		calls++
		if calls == 1 {
			return WordDetails{}, errors.New("temporary outage")
		}
		return WordDetails{Word: "casa"}, nil
	}), nil, nil)
	if _, err := s.Preview(context.Background(), "casa"); err == nil {
		t.Fatal("expected source failure")
	}
	if result, err := s.Preview(context.Background(), "casa"); err != nil || result.Details.Word != "casa" {
		t.Fatalf("failure was cached: %v", err)
	}
}

func (f *fakeWordSource) Lookup(_ context.Context, word string) (WordDetails, error) {
	f.word = word
	return f.details, nil
}

type fakeLessonGenerator struct {
	lesson *LearningGuide
	err    error
	calls  int
}

func (f *fakeLessonGenerator) Generate(context.Context, WordDetails) (*LearningGuide, error) {
	f.calls++
	return f.lesson, f.err
}

func TestServiceNormalizesWordAndCachesLesson(t *testing.T) {
	source := &fakeWordSource{details: WordDetails{Word: "parlare"}}
	lesson := &LearningGuide{
		Meaning:     "to speak",
		Summary:     "A regular verb",
		GrammarNote: "Uses avere",
		Examples: []LearningExample{
			{Italian: "Parlo.", English: "I speak."},
			{Italian: "Parli.", English: "You speak."},
			{Italian: "Parliamo.", English: "We speak."},
			{Italian: "Parlano.", English: "They speak."},
		},
	}
	teacher := &fakeLessonGenerator{lesson: lesson}
	service := NewService(source, teacher, NewMemoryLessonCache())

	for range 2 {
		result, err := service.Lookup(context.Background(), "  PARLARE ")
		if err != nil {
			t.Fatal(err)
		}
		if result.Details.Learning != lesson {
			t.Fatalf("missing lesson: %+v", result.Details)
		}
	}
	if source.word != "parlare" || teacher.calls != 1 {
		t.Fatalf("word=%q teacher calls=%d", source.word, teacher.calls)
	}
}

func TestServiceReturnsLessonFailureAsWarning(t *testing.T) {
	source := &fakeWordSource{details: WordDetails{Word: "casa"}}
	teacher := &fakeLessonGenerator{err: errors.New("provider unavailable")}
	service := NewService(source, teacher, NewMemoryLessonCache())
	for range 2 {
		result, err := service.Lookup(context.Background(), "casa")
		if err != nil {
			t.Fatal(err)
		}
		if result.LessonWarning == nil || result.Details.LearningError == "" {
			t.Fatalf("expected lesson warning, got %+v", result)
		}
	}
	if teacher.calls != 2 {
		t.Fatalf("failed response was cached; teacher calls=%d", teacher.calls)
	}
}

func TestServiceDoesNotCacheIncompleteLesson(t *testing.T) {
	source := &fakeWordSource{details: WordDetails{Word: "casa"}}
	teacher := &fakeLessonGenerator{lesson: &LearningGuide{Meaning: "house"}}
	service := NewService(source, teacher, NewMemoryLessonCache())

	for range 2 {
		result, err := service.Lookup(context.Background(), "casa")
		if err != nil {
			t.Fatal(err)
		}
		if result.LessonWarning == nil || result.Details.Learning != nil {
			t.Fatalf("expected rejected incomplete lesson, got %+v", result)
		}
	}
	if teacher.calls != 2 {
		t.Fatalf("incomplete response was cached; teacher calls=%d", teacher.calls)
	}
}

func TestNormalizeWordRejectsInvalidInput(t *testing.T) {
	if _, err := NormalizeWord(" "); !errors.Is(err, ErrInvalidWord) {
		t.Fatalf("expected ErrInvalidWord, got %v", err)
	}
}
