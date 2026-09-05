// Package skills embeds the agent skill shipped with Nummion.
package skills

import "embed"

// FS contains the Lexware agent skill exactly as installed by the CLI.
//
//go:embed lexware
var FS embed.FS
