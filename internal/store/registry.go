package store

import (
	"fmt"
	"log"
	"sort"

	"github.com/rogersnm/compass/internal/config"
	"github.com/rogersnm/compass/internal/id"
)

// Registry routes commands to the correct Store based on project key.
type Registry struct {
	stores       map[string]Store // "local" and/or store names -> Store
	defaultStore string           // "local" or a store name
	cfg          *config.Config
	dataDir      string
}

func NewRegistry(cfg *config.Config, dataDir string) *Registry {
	return &Registry{
		stores:       make(map[string]Store),
		defaultStore: cfg.DefaultStore,
		cfg:          cfg,
		dataDir:      dataDir,
	}
}

func (r *Registry) Add(name string, s Store) {
	r.stores[name] = s
}

// Get returns a store by name.
func (r *Registry) Get(name string) (Store, error) {
	s, ok := r.stores[name]
	if !ok {
		return nil, fmt.Errorf("store %q not configured", name)
	}
	return s, nil
}

// DefaultStore returns the default store and its name.
func (r *Registry) Default() (Store, string, error) {
	if r.defaultStore == "" {
		return nil, "", fmt.Errorf("no default store configured")
	}
	s, err := r.Get(r.defaultStore)
	if err != nil {
		return nil, "", err
	}
	return s, r.defaultStore, nil
}

// ForProject resolves a project key to its store using the cached mapping.
// On cache miss, probes all stores (local first).
func (r *Registry) ForProject(projectKey string) (Store, string, error) {
	if r.cfg.Projects != nil {
		if storeName, ok := r.cfg.Projects[projectKey]; ok {
			s, err := r.Get(storeName)
			if err == nil {
				if _, _, err := s.GetProject(projectKey); err == nil {
					return s, storeName, nil
				}
			}
			// Stale cache entry; prune it
			r.UncacheProject(projectKey)
		}
	}

	// Cache miss: probe all stores, local first
	for _, name := range r.probeOrder() {
		s := r.stores[name]
		if _, _, err := s.GetProject(projectKey); err == nil {
			r.CacheProject(projectKey, name)
			return s, name, nil
		}
	}

	return nil, "", fmt.Errorf("project %s not found on any configured store", projectKey)
}

// ForEntity extracts the project key from an entity ID and routes to its store.
func (r *Registry) ForEntity(entityID string) (Store, string, error) {
	key, err := id.ProjectKeyFrom(entityID)
	if err != nil {
		return nil, "", err
	}
	return r.ForProject(key)
}

// All returns all configured stores.
func (r *Registry) All() map[string]Store {
	return r.stores
}

// CloudStoreNames returns sorted names of cloud stores.
func (r *Registry) CloudStoreNames() []string {
	var names []string
	for name := range r.stores {
		if name != "local" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// CacheProject writes the project-to-store mapping and persists config.
func (r *Registry) CacheProject(key, storeName string) {
	if r.cfg.Projects == nil {
		r.cfg.Projects = make(map[string]string)
	}
	r.cfg.Projects[key] = storeName
	if err := config.Save(r.dataDir, r.cfg); err != nil {
		log.Printf("warning: failed to persist project cache: %v", err)
	}
}

// UncacheProject removes a project from the cache and persists.
func (r *Registry) UncacheProject(key string) {
	if r.cfg.Projects == nil {
		return
	}
	delete(r.cfg.Projects, key)
	if err := config.Save(r.dataDir, r.cfg); err != nil {
		log.Printf("warning: failed to persist project cache: %v", err)
	}
}

// probeOrder returns store names with "local" first.
func (r *Registry) probeOrder() []string {
	var names []string
	if _, ok := r.stores["local"]; ok {
		names = append(names, "local")
	}
	for _, n := range r.CloudStoreNames() {
		names = append(names, n)
	}
	return names
}

// IsEmpty returns true when no stores are registered.
func (r *Registry) IsEmpty() bool {
	return len(r.stores) == 0
}

// Names returns all store names.
func (r *Registry) Names() []string {
	var names []string
	for n := range r.stores {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// SetDefault changes the default store.
func (r *Registry) SetDefault(name string) {
	r.defaultStore = name
}

// DefaultName returns the default store name.
func (r *Registry) DefaultName() string {
	return r.defaultStore
}
