package main

import (
	"context"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nodo/internal/database"
)

const (
	defaultProfileBaseURL = "https://www.unistrapg.it/profilo_lingua_italiana/site"
	sourceName            = "profilo-lingua-italiana"
	legacySourceName      = "kelly-project-italian"
)

var (
	entryPattern = regexp.MustCompile(`(?is)<a\b[^>]*>([^<]+)</a>\s*\(([^)]*)\)`)
	lemmaPattern = regexp.MustCompile(`^[\p{L}][\p{L}'’-]*$`)
)

type curriculumWord struct {
	Word         string
	Level        string
	PartOfSpeech string
	Rank         int
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}
	words, err := downloadCurriculum(ctx, env("ITALIAN_PROFILE_BASE_URL", defaultProfileBaseURL))
	if err != nil {
		log.Fatal(err)
	}
	counts, err := replaceWords(ctx, pool, words)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Imported Profilo della lingua italiana words: A1=%d A2=%d B1=%d B2=%d\n", counts["A1"], counts["A2"], counts["B1"], counts["B2"])
}

func downloadCurriculum(ctx context.Context, baseURL string) ([]curriculumWord, error) {
	client := &http.Client{Timeout: 45 * time.Second}
	levels := []string{"A1", "A2", "B1", "B2"}
	seen := make(map[string]bool)
	var result []curriculumWord
	for _, level := range levels {
		url := strings.TrimRight(baseURL, "/") + "/liste_lessicali_" + strings.ToLower(level) + ".html"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("download %s vocabulary: %w", level, err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read %s vocabulary: %w", level, readErr)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("download %s vocabulary: status %d", level, resp.StatusCode)
		}
		parsed := parseCurriculumPage(body, level)
		if len(parsed) == 0 {
			return nil, fmt.Errorf("parse %s vocabulary: no nouns or verbs found", level)
		}
		for _, item := range parsed {
			key := item.Word + "\x00" + item.PartOfSpeech
			if seen[key] {
				continue
			}
			seen[key] = true
			result = append(result, item)
		}
	}
	return result, nil
}

func parseCurriculumPage(body []byte, level string) []curriculumWord {
	matches := entryPattern.FindAllSubmatch(body, -1)
	result := make([]curriculumWord, 0, len(matches))
	for index, match := range matches {
		partOfSpeech := normalizePartOfSpeech(string(match[2]))
		if partOfSpeech == "" {
			continue
		}
		for _, word := range normalizeLemmas(string(match[1])) {
			result = append(result, curriculumWord{Word: word, Level: level, PartOfSpeech: partOfSpeech, Rank: index + 1})
		}
	}
	return result
}

func normalizePartOfSpeech(value string) string {
	value = strings.ToLower(strings.TrimSpace(html.UnescapeString(value)))
	switch {
	case strings.HasPrefix(value, "s."):
		return "n"
	case strings.HasPrefix(value, "v."):
		return "v"
	default:
		return ""
	}
}

func normalizeLemmas(value string) []string {
	value = strings.ToLower(strings.TrimSpace(html.UnescapeString(value)))
	value = strings.ReplaceAll(value, "’", "'")
	var candidates []string
	if open := strings.IndexByte(value, '('); open > 0 && strings.HasSuffix(value, ")") {
		candidates = append(candidates, value[:open], value[open+1:len(value)-1])
	} else {
		candidates = append(candidates, value)
	}
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if before, _, found := strings.Cut(candidate, "/"); found {
			candidate = strings.TrimSpace(before)
		}
		if lemmaPattern.MatchString(candidate) {
			result = append(result, candidate)
		}
	}
	return result
}

func replaceWords(ctx context.Context, pool *pgxpool.Pool, words []curriculumWord) (map[string]int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE words SET active=false WHERE source IN ($1,$2)`, sourceName, legacySourceName); err != nil {
		return nil, err
	}
	for _, item := range words {
		var id int64
		err := tx.QueryRow(ctx, `SELECT id FROM words WHERE word=$1 AND part_of_speech=$2 ORDER BY (source=$3) DESC, active DESC, id LIMIT 1`, item.Word, item.PartOfSpeech, sourceName).Scan(&id)
		switch {
		case err == nil:
			_, err = tx.Exec(ctx, `UPDATE words SET level=$2,frequency_per_million=0,curriculum_rank=$3,source=$4,active=true WHERE id=$1`, id, item.Level, item.Rank, sourceName)
		case err == pgx.ErrNoRows:
			_, err = tx.Exec(ctx, `INSERT INTO words(word,level,part_of_speech,frequency_per_million,curriculum_rank,source,active) VALUES($1,$2,$3,0,$4,$5,true)`, item.Word, item.Level, item.PartOfSpeech, item.Rank, sourceName)
		}
		if err != nil {
			return nil, fmt.Errorf("save %s (%s): %w", item.Word, item.PartOfSpeech, err)
		}
	}
	counts := map[string]int{"A1": 0, "A2": 0, "B1": 0, "B2": 0}
	rows, err := tx.Query(ctx, `SELECT level,count(*) FROM words WHERE source=$1 AND active GROUP BY level`, sourceName)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			rows.Close()
			return nil, err
		}
		counts[level] = count
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return counts, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
