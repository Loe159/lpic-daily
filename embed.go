// Package lpicdaily exposes immutable built-in project assets.
//
// Application packages should treat BuiltinFS as read-only. The files under
// curriculum/ and schemas/ remain the repository source of truth.
package lpicdaily

import "embed"

// BuiltinFS contains the built-in LPIC curriculum and content contracts.
//
//go:embed curriculum/lpic-1-v5/*.json schemas/*.schema.json
var BuiltinFS embed.FS
