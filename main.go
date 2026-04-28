package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"misskeyRSSbot/internal/application"
	"misskeyRSSbot/internal/domain/repository"
	"misskeyRSSbot/internal/infrastructure/llm"
	"misskeyRSSbot/internal/infrastructure/misskey"
	"misskeyRSSbot/internal/infrastructure/rss"
	"misskeyRSSbot/internal/infrastructure/storage"
	"misskeyRSSbot/internal/interfaces/config"
)

func main() {
	fmt.Println("Starting Misskey RSS Bot...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	feedRepo := rss.NewFeedRepository()
	noteRepo := misskey.NewNoteRepository(misskey.Config{
		Host:           cfg.MisskeyHost,
		AuthToken:      cfg.AuthToken,
		MaxPermits:     cfg.MaxPermits,
		RefillInterval: cfg.GetRefillInterval(),
		LocalOnly:      cfg.LocalOnly,
	})

	type cacheWithCleanup interface {
		CleanupOldGUIDs(ctx context.Context, olderThan time.Duration) (int64, error)
	}

	var cacheRepo repository.CacheRepository
	var cacheCloser io.Closer
	var cacheCleaner cacheWithCleanup
	firstRunLatestOnly := cfg.FirstRunLatestOnly

	if cfg.IsPersistentCache() {
		sqliteCache, cacheErr := storage.NewSQLiteCacheRepository(cfg.CacheDBPath)
		if cacheErr == nil {
			cacheRepo = sqliteCache
			if closer, ok := sqliteCache.(io.Closer); ok {
				cacheCloser = closer
			}
			if cleaner, ok := sqliteCache.(cacheWithCleanup); ok {
				cacheCleaner = cleaner
			}
			log.Printf("Using persistent cache: %s", cfg.CacheDBPath)
		} else {
			log.Printf("Warning: Failed to initialize SQLite cache: %v", cacheErr)
			log.Println("Falling back to in-memory cache")
		}
	}

	if cacheRepo == nil {
		cacheRepo = storage.NewMemoryCacheRepository()
		log.Println("Using in-memory cache (data will not persist across restarts)")
		if !cfg.FirstRunLatestOnly {
			log.Printf("Warning: FIRST_RUN_LATEST_ONLY=%v requires CACHE_DB_PATH to be set when using in-memory cache", cfg.FirstRunLatestOnly)
			log.Printf("Overriding FIRST_RUN_LATEST_ONLY from %v to true for safety", cfg.FirstRunLatestOnly)
			firstRunLatestOnly = true
		}
	}

	llmCfg := cfg.GetLLMConfig()
	summarizerRepo, err := llm.NewSummarizerRepository(ctx, llm.Config{
		Provider:          llmCfg.Provider,
		APIKey:            llmCfg.APIKey,
		Model:             llmCfg.Model,
		Region:            llmCfg.Region,
		MaxTokens:         llmCfg.MaxTokens,
		Timeout:           llmCfg.Timeout,
		SystemInstruction: llmCfg.SystemInstruction,
	})
	if err != nil {
		log.Printf("Warning: LLM summarizer initialization failed: %v", err)
		log.Println("Attempting fallback to noop summarizer...")
		summarizerRepo, err = llm.NewSummarizerRepository(ctx, llm.Config{Provider: "noop"})
		if err != nil {
			log.Printf("Warning: noop summarizer initialization failed unexpectedly: %v", err)
			log.Println("Continuing without summarization feature")
		}
	}

	service := application.NewRSSFeedService(
		feedRepo,
		noteRepo,
		cacheRepo,
		summarizerRepo,
		application.WithFirstRunLatestOnly(firstRunLatestOnly),
	)

	if firstRunLatestOnly {
		log.Println("First run mode: post latest entry only")
	} else {
		log.Println("First run mode: post all unprocessed entries")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutdown signal received")
		cancel()
	}()

	interval := cfg.GetFetchInterval()
	log.Printf("RSS fetch interval: %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var cleanupTicker *time.Ticker
	if cacheCleaner != nil {
		cleanupInterval := cfg.GetCacheCleanupInterval()
		log.Printf("Cache cleanup interval: %v", cleanupInterval)
		cleanupTicker = time.NewTicker(cleanupInterval)
		defer cleanupTicker.Stop()
	}

	log.Println("Fetching RSS feeds...")
	if err := service.ProcessAllFeeds(ctx, cfg.RSSURL); err != nil {
		log.Printf("RSS processing error: %v", err)
	}
	log.Println("RSS feeds fetched")

	cleanupChan := func() <-chan time.Time {
		if cleanupTicker != nil {
			return cleanupTicker.C
		}
		return nil
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down...")
			if cacheCloser != nil {
				if err := cacheCloser.Close(); err != nil {
					log.Printf("Failed to close cache: %v", err)
				}
			}
			return
		case <-ticker.C:
			log.Println("Fetching RSS feeds...")
			if err := service.ProcessAllFeeds(ctx, cfg.RSSURL); err != nil {
				log.Printf("RSS processing error: %v", err)
			}
			log.Println("RSS feeds fetched")
		case <-cleanupChan:
			retentionPeriod := cfg.GetCacheRetentionPeriod()
			deleted, err := cacheCleaner.CleanupOldGUIDs(ctx, retentionPeriod)
			if err != nil {
				log.Printf("Cache cleanup error: %v", err)
			} else if deleted > 0 {
				log.Printf("Cache cleanup: removed %d old entries", deleted)
			}
		}
	}
}
