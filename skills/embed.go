// Package skills embeds the agent skill shipped with Nummion.
package skills

import "embed"

// FS contains the Nummion agent skill exactly as installed by the CLI.
//
//go:embed nummion
var FS embed.FS
