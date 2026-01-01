package web

import "embed"

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templatesFS embed.FS
