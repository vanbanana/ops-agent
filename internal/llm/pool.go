package llm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Provider represents a configured LLM provider entry in the model pool.
type Provider struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ProviderName  string `json:"provider"`
	BaseURL       string `json:"base_url"`
	APIKey        string `json:"api_key"`
	ModelID       string `json:"model_id"`
	ContextWindow int    `json:"context_window"`
	MaxOutput     int    `json:"max_output"`
	IsActive      bool   `json:"is_active"`
	CanReason     bool   `json:"can_reason"`
}

// ProviderPublic is a Provider with api_key masked for API responses.
type ProviderPublic struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ProviderName  string `json:"provider"`
	BaseURL       string `json:"base_url"`
	APIKeyMasked  string `json:"api_key_masked"`
	ModelID       string `json:"model_id"`
	ContextWindow int    `json:"context_window"`
	MaxOutput     int    `json:"max_output"`
	IsActive      bool   `json:"is_active"`
	CanReason     bool   `json:"can_reason"`
}

// MaskAPIKey returns a masked version showing first 4 and last 4 chars.
func MaskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// ToPublic converts a Provider to its public (masked) form.
func (p *Provider) ToPublic() ProviderPublic {
	return ProviderPublic{
		ID:            p.ID,
		Name:          p.Name,
		ProviderName:  p.ProviderName,
		BaseURL:       p.BaseURL,
		APIKeyMasked:  MaskAPIKey(p.APIKey),
		ModelID:       p.ModelID,
		ContextWindow: p.ContextWindow,
		MaxOutput:     p.MaxOutput,
		IsActive:      p.IsActive,
		CanReason:     p.CanReason,
	}
}

// ModelPool manages the set of configured providers and the active selection.
type ModelPool struct {
	mu        sync.RWMutex
	providers []Provider
	db        *sql.DB
	onChange  func(active Provider) // callback when active model changes
}

// NewModelPool creates a model pool. Load order:
// 1. If DB has "model_pool" config key, use that.
// 2. Else if providersFile exists, load from JSON file.
// 3. Else create a default entry from env vars.
func NewModelPool(db *sql.DB, providersFile string, fallbackBaseURL, fallbackAPIKey, fallbackModel string) *ModelPool {
	pool := &ModelPool{db: db}

	// Try loading from DB first
	if db != nil {
		var val string
		err := db.QueryRow("SELECT value FROM configs WHERE key = 'model_pool'").Scan(&val)
		if err == nil && val != "" {
			var providers []Provider
			if json.Unmarshal([]byte(val), &providers) == nil && len(providers) > 0 {
				pool.providers = providers
				pool.ensureOneActive()
				return pool
			}
		}
	}

	// Try loading from file
	if providersFile != "" {
		data, err := os.ReadFile(providersFile)
		if err == nil {
			var providers []Provider
			if json.Unmarshal(data, &providers) == nil && len(providers) > 0 {
				pool.providers = providers
				pool.ensureOneActive()
				pool.persist()
				return pool
			}
		}
	}

	// Fallback: create from env
	if fallbackBaseURL != "" && fallbackAPIKey != "" {
		name := fallbackModel
		if info, ok := SupportedModels[fallbackModel]; ok {
			name = info.Name
		}
		pool.providers = []Provider{{
			ID:            fallbackModel,
			Name:          name,
			ProviderName:  "default",
			BaseURL:       fallbackBaseURL,
			APIKey:        fallbackAPIKey,
			ModelID:       fallbackModel,
			ContextWindow: GetModelInfo(fallbackModel).ContextWindow,
			MaxOutput:     GetModelInfo(fallbackModel).MaxOutput,
			IsActive:      true,
			CanReason:     GetModelInfo(fallbackModel).CanReason,
		}}
		pool.persist()
	}

	return pool
}

// SetOnChange registers a callback fired when the active model switches.
func (p *ModelPool) SetOnChange(fn func(active Provider)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChange = fn
}

// GetAll returns all providers (public view with masked keys).
func (p *ModelPool) GetAll() []ProviderPublic {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]ProviderPublic, len(p.providers))
	for i, prov := range p.providers {
		result[i] = prov.ToPublic()
	}
	return result
}

// GetActive returns the currently active provider (full, with key).
func (p *ModelPool) GetActive() (Provider, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, prov := range p.providers {
		if prov.IsActive {
			return prov, true
		}
	}
	if len(p.providers) > 0 {
		return p.providers[0], true
	}
	return Provider{}, false
}

// GetActivePublic returns the active provider in public (masked) form.
func (p *ModelPool) GetActivePublic() (ProviderPublic, bool) {
	active, ok := p.GetActive()
	if !ok {
		return ProviderPublic{}, false
	}
	return active.ToPublic(), true
}

// SavePool replaces the entire pool with new providers and persists.
// If a provider has an empty API key but shares base_url with an existing provider, the key is inherited.
func (p *ModelPool) SavePool(providers []Provider) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Build lookup of existing keys by base_url for key inheritance
	keyByURL := map[string]string{}
	for _, existing := range p.providers {
		if existing.APIKey != "" {
			keyByURL[existing.BaseURL] = existing.APIKey
		}
	}
	// Also build lookup by ID for edit-without-changing-key scenario
	keyByID := map[string]string{}
	for _, existing := range p.providers {
		if existing.APIKey != "" {
			keyByID[existing.ID] = existing.APIKey
		}
	}

	for i := range providers {
		if providers[i].APIKey == "" {
			// First try to inherit from same ID (edit case)
			if k, ok := keyByID[providers[i].ID]; ok {
				providers[i].APIKey = k
			} else if k, ok := keyByURL[providers[i].BaseURL]; ok {
				// Inherit from same base_url (same provider, different model)
				providers[i].APIKey = k
			}
		}
	}

	p.providers = providers
	p.ensureOneActive()
	return p.persistLocked()
}

// Switch sets the active provider by ID. Returns error if not found.
func (p *ModelPool) Switch(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	found := false
	var active Provider
	for i := range p.providers {
		if p.providers[i].ID == id {
			p.providers[i].IsActive = true
			active = p.providers[i]
			found = true
		} else {
			p.providers[i].IsActive = false
		}
	}
	if !found {
		return fmt.Errorf("provider %q not found in pool", id)
	}

	if err := p.persistLocked(); err != nil {
		return err
	}

	if p.onChange != nil {
		p.onChange(active)
	}
	return nil
}

// GetProviderByID returns a full provider by ID (including API key).
func (p *ModelPool) GetProviderByID(id string) (Provider, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, prov := range p.providers {
		if prov.ID == id {
			return prov, true
		}
	}
	return Provider{}, false
}

func (p *ModelPool) ensureOneActive() {
	hasActive := false
	for _, prov := range p.providers {
		if prov.IsActive {
			hasActive = true
			break
		}
	}
	if !hasActive && len(p.providers) > 0 {
		p.providers[0].IsActive = true
	}
}

func (p *ModelPool) persist() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.persistLocked()
}

func (p *ModelPool) persistLocked() error {
	if p.db == nil {
		return nil
	}
	data, err := json.Marshal(p.providers)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(
		`INSERT OR REPLACE INTO configs (key, value, updated_at) VALUES ('model_pool', ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		string(data),
	)
	return err
}
