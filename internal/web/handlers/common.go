package handlers

import (
	"log"
	"net/http"
	"strings"
)

// formatServiceName formats a service name for display (converts kebab-case to Title Case)
func formatServiceName(name string) string {
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

// httpError logs the full error internally and returns a generic message to the client.
// This prevents exposing internal error details to users.
func httpError(w http.ResponseWriter, context string, err error, statusCode int) {
	log.Printf("Error [%s]: %v", context, err)
	http.Error(w, "An internal error occurred. Please try again later.", statusCode)
}
