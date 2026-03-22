package book

import "embed"

// StaticFiles contains the compiled SvelteKit UI.
// Built by: cd web && npm run build && cp -r build/ ../static/
//
//go:embed all:static
var StaticFiles embed.FS
