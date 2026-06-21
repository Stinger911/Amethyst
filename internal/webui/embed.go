// Package webui embeds the built React frontend (web/) into the Go
// binary so amethyst can serve both the API and the UI from one process,
// with no separate web server needed for self-hosted deploys.
package webui

import "embed"

// DistFS embeds web/'s vite build output. vite.config.ts points its
// build.outDir at this dist/ directory directly, so `npm run build`
// writes here. Until that's been run, dist/ only contains placeholder.txt
// (kept so the go:embed directive below always has something to embed).
//
//go:embed dist
var DistFS embed.FS
