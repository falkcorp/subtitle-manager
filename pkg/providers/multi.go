// file: pkg/providers/multi.go
// version: 1.1.0
// guid: 2f8c1a05-6d43-4b79-9e20-3a7c5d8b0164

package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/events"
)

// FetchFromAll tries each known provider in order until one returns a subtitle.
// It uses an increasing delay between provider attempts to avoid rapid retries.
// The provider API key is reused when applicable. The name of the provider that
// succeeded is returned along with the subtitle bytes.
func FetchFromAll(ctx context.Context, mediaPath, lang, key string) ([]byte, string, error) {
	insts := Instances()
	if len(insts) == 0 {
		names := All()
		delay := time.Second
		for i, name := range names {
			p, err := Get(name, key)
			if err != nil {
				continue
			}
			c, cancel := context.WithTimeout(ctx, 15*time.Second)
			data, err := p.Fetch(c, mediaPath, lang)
			cancel()
			if err == nil {
				return data, name, nil
			}
			if ctx.Err() != nil {
				return nil, "", ctx.Err()
			}
			// Context-aware inter-provider delay so a cancelled or bounded
			// context aborts immediately instead of blocking on a blind sleep
			// (this matches the instance-based loops below).
			select {
			case <-time.After(time.Duration(i+1) * delay):
			case <-ctx.Done():
				return nil, "", ctx.Err()
			}
		}
		return nil, "", fmt.Errorf("no subtitle found")
	}

	delay := time.Second
	var lastError error
	for i, inst := range insts {
		if IsInBackoff(inst.ID) {
			continue
		}
		p, err := Get(inst.Name, key)
		if err != nil {
			continue
		}
		c, cancel := context.WithTimeout(ctx, 15*time.Second)
		data, err := p.Fetch(c, mediaPath, lang)
		cancel()
		if err == nil {
			SetBackoff(inst.ID, 0)
			return data, inst.ID, nil
		}
		lastError = err
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		SetBackoff(inst.ID, time.Duration(i+1)*delay)
		select {
		case <-time.After(time.Duration(i+1) * delay):
		case <-ctx.Done():
			return nil, "", ctx.Err()
		}
	}

	// Send search failed event
	errorMsg := "no subtitle found"
	if lastError != nil {
		errorMsg = lastError.Error()
	}
	events.PublishSearchFailed(ctx, events.SearchFailedData{
		Query:     fmt.Sprintf("media:%s lang:%s", mediaPath, lang),
		Language:  lang,
		Error:     errorMsg,
		Timestamp: time.Now(),
	})

	return nil, "", fmt.Errorf("no subtitle found")
}

// FetchFromTagged limits provider attempts to instances matching all tags.
func FetchFromTagged(ctx context.Context, mediaPath, lang, key string, tags []string, tm interface {
	FilterByTags(string, []string) ([]string, error)
}) ([]byte, string, error) {
	insts, err := InstancesByTags(tm, tags)
	if err != nil {
		return nil, "", err
	}
	if len(insts) == 0 {
		return nil, "", fmt.Errorf("no subtitle found")
	}
	return fetchFromInstances(ctx, insts, mediaPath, lang, key)
}

func fetchFromInstances(ctx context.Context, insts []Instance, mediaPath, lang, key string) ([]byte, string, error) {
	delay := time.Second
	for i, inst := range insts {
		if IsInBackoff(inst.ID) {
			continue
		}
		p, err := Get(inst.Name, key)
		if err != nil {
			continue
		}
		c, cancel := context.WithTimeout(ctx, 15*time.Second)
		data, err := p.Fetch(c, mediaPath, lang)
		cancel()
		if err == nil {
			SetBackoff(inst.ID, 0)
			return data, inst.ID, nil
		}
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		SetBackoff(inst.ID, time.Duration(i+1)*delay)
		select {
		case <-time.After(time.Duration(i+1) * delay):
		case <-ctx.Done():
			return nil, "", ctx.Err()
		}
	}
	return nil, "", fmt.Errorf("no subtitle found")
}
