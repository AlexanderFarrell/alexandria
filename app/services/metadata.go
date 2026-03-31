package services

import (
	"context"
	"time"

	"alexandria/app/ports"
)

// ProviderError records a failure from one metadata provider.
type ProviderError struct {
	Provider string `json:"provider"`
	Message  string `json:"message"`
}

// MetadataService fans search queries out to all registered providers concurrently.
type MetadataService struct {
	providers []ports.MetadataProvider
	timeout   time.Duration
}

func NewMetadataService(providers []ports.MetadataProvider) *MetadataService {
	return &MetadataService{
		providers: providers,
		timeout:   5 * time.Second,
	}
}

type providerResult struct {
	providerID string
	results    []ports.MetadataResult
	err        error
}

// Search queries all active providers concurrently and aggregates results.
// It always returns HTTP 200-level data; individual provider failures are
// recorded in errs rather than aborting the whole search.
func (s *MetadataService) Search(
	ctx context.Context,
	q ports.MetadataQuery,
) (results []ports.MetadataResult, errs []ProviderError, err error) {
	if len(s.providers) == 0 {
		return nil, nil, nil
	}

	tctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Buffered so goroutines can send without blocking if we time out
	ch := make(chan providerResult, len(s.providers))

	for _, p := range s.providers {
		p := p
		go func() {
			res, e := p.Search(tctx, q)
			ch <- providerResult{providerID: p.ID(), results: res, err: e}
		}()
	}

	for range s.providers {
		pr := <-ch
		if pr.err != nil {
			errs = append(errs, ProviderError{Provider: pr.providerID, Message: pr.err.Error()})
		} else {
			results = append(results, pr.results...)
		}
	}

	return results, errs, nil
}
