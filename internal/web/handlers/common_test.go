package handlers

import (
	"testing"
)

func TestGroupByCategory(t *testing.T) {
	serviceMap := map[string]ServiceInfo{
		"plex":         {Name: "plex", Category: "media"},
		"radarr":       {Name: "radarr", Category: "media"},
		"qbittorrent":  {Name: "qbittorrent", Category: "downloads"},
		"traefik":      {Name: "traefik", Category: "networking"},
		"nocategory":   {Name: "nocategory", Category: ""},
	}

	result := groupByCategory(serviceMap)

	if len(result["media"]) != 2 {
		t.Errorf("expected 2 media services, got %d", len(result["media"]))
	}
	if len(result["downloads"]) != 1 {
		t.Errorf("expected 1 downloads service, got %d", len(result["downloads"]))
	}
	if len(result["networking"]) != 1 {
		t.Errorf("expected 1 networking service, got %d", len(result["networking"]))
	}
	if len(result["other"]) != 1 {
		t.Errorf("expected empty category mapped to 'other', got %d", len(result["other"]))
	}
}

func TestGroupByCategoryEmpty(t *testing.T) {
	result := groupByCategory(map[string]ServiceInfo{})
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d categories", len(result))
	}
}

