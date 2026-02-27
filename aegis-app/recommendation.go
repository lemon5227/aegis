package main

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

// RecommendationStrategy defines the interface for content recommendation algorithms
type RecommendationStrategy interface {
	Name() string
	// Rank takes a list of candidate posts and returns them sorted by score
	Rank(candidates []ForumMessage, viewerPubkey string, now int64) ([]FeedStreamItem, error)
}

// StrategyRegistry manages available recommendation strategies
type StrategyRegistry struct {
	mu         sync.RWMutex
	strategies map[string]RecommendationStrategy
}

var globalStrategyRegistry = &StrategyRegistry{
	strategies: make(map[string]RecommendationStrategy),
}

func RegisterStrategy(strategy RecommendationStrategy) {
	globalStrategyRegistry.mu.Lock()
	defer globalStrategyRegistry.mu.Unlock()
	globalStrategyRegistry.strategies[strings.ToLower(strategy.Name())] = strategy
}

func GetStrategy(name string) (RecommendationStrategy, error) {
	globalStrategyRegistry.mu.RLock()
	defer globalStrategyRegistry.mu.RUnlock()

	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "hot-v1"
	}

	if strategy, ok := globalStrategyRegistry.strategies[name]; ok {
		return strategy, nil
	}

	// Fallback to hot-v1 if registered, otherwise error
	if strategy, ok := globalStrategyRegistry.strategies["hot-v1"]; ok {
		return strategy, nil
	}

	return nil, errors.New("strategy not found and fallback unavailable")
}

// HotV1Strategy implements the classic Hacker News / Reddit style gravity decay
type HotV1Strategy struct{}

func (s *HotV1Strategy) Name() string {
	return "hot-v1"
}

func (s *HotV1Strategy) Rank(candidates []ForumMessage, viewerPubkey string, now int64) ([]FeedStreamItem, error) {
	items := make([]FeedStreamItem, 0, len(candidates))

	for _, post := range candidates {
		score := computeHotScore(post.Score, post.Timestamp, now)
		items = append(items, FeedStreamItem{
			Post:                post,
			Reason:              "recommended_hot",
			IsSubscribed:        false,
			RecommendationScore: score,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RecommendationScore == items[j].RecommendationScore {
			return items[i].Post.Timestamp > items[j].Post.Timestamp
		}
		return items[i].RecommendationScore > items[j].RecommendationScore
	})

	return items, nil
}

// NewStrategy implements a simple time-based sorting
type NewStrategy struct{}

func (s *NewStrategy) Name() string {
	return "new"
}

func (s *NewStrategy) Rank(candidates []ForumMessage, viewerPubkey string, now int64) ([]FeedStreamItem, error) {
	items := make([]FeedStreamItem, 0, len(candidates))

	for _, post := range candidates {
		// Score is just timestamp
		items = append(items, FeedStreamItem{
			Post:                post,
			Reason:              "recommended_new",
			IsSubscribed:        false,
			RecommendationScore: float64(post.Timestamp),
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Post.Timestamp > items[j].Post.Timestamp
	})

	return items, nil
}

func init() {
	RegisterStrategy(&HotV1Strategy{})
	RegisterStrategy(&NewStrategy{})
}
