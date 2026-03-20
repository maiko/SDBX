package handlers

import (
	"testing"
)

func TestGroupByCategory(t *testing.T) {
	serviceMap := map[string]ServiceInfo{
		"plex":        {Name: "plex", Category: "media"},
		"radarr":      {Name: "radarr", Category: "media"},
		"qbittorrent": {Name: "qbittorrent", Category: "downloads"},
		"traefik":     {Name: "traefik", Category: "networking"},
		"nocategory":  {Name: "nocategory", Category: ""},
	}

	result := groupByCategory(serviceMap)

	// Build a lookup for easier assertions
	lookup := make(map[string][]ServiceInfo)
	for _, group := range result {
		lookup[group.Name] = group.Services
	}

	if len(lookup["media"]) != 2 {
		t.Errorf("expected 2 media services, got %d", len(lookup["media"]))
	}
	if len(lookup["downloads"]) != 1 {
		t.Errorf("expected 1 downloads service, got %d", len(lookup["downloads"]))
	}
	if len(lookup["networking"]) != 1 {
		t.Errorf("expected 1 networking service, got %d", len(lookup["networking"]))
	}
	if len(lookup["other"]) != 1 {
		t.Errorf("expected empty category mapped to 'other', got %d", len(lookup["other"]))
	}
}

func TestGroupByCategoryEmpty(t *testing.T) {
	result := groupByCategory(map[string]ServiceInfo{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d categories", len(result))
	}
}

func TestGroupByCategoryStableOrder(t *testing.T) {
	serviceMap := map[string]ServiceInfo{
		"traefik":     {Name: "traefik", Category: "networking"},
		"authelia":    {Name: "authelia", Category: "auth"},
		"plex":        {Name: "plex", Category: "media"},
		"qbittorrent": {Name: "qbittorrent", Category: "downloads"},
	}

	result := groupByCategory(serviceMap)

	expectedOrder := []string{"media", "downloads", "auth", "networking"}
	if len(result) != len(expectedOrder) {
		t.Fatalf("expected %d groups, got %d", len(expectedOrder), len(result))
	}
	for i, expected := range expectedOrder {
		if result[i].Name != expected {
			t.Errorf("group[%d] = %q, want %q", i, result[i].Name, expected)
		}
	}
}

func TestGroupByCategoryUnknownCategory(t *testing.T) {
	serviceMap := map[string]ServiceInfo{
		"custom": {Name: "custom", Category: "custom-cat"},
		"plex":   {Name: "plex", Category: "media"},
	}

	result := groupByCategory(serviceMap)

	if len(result) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result))
	}
	// media should come first (in CategoryOrder), custom-cat appended at end
	if result[0].Name != "media" {
		t.Errorf("first group = %q, want 'media'", result[0].Name)
	}
	if result[1].Name != "custom-cat" {
		t.Errorf("second group = %q, want 'custom-cat'", result[1].Name)
	}
}
