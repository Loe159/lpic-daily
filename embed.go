// Package lpicdaily exposes immutable built-in project assets.
//
// Application packages should treat BuiltinFS as read-only. The files under
// curriculum/, schemas/ and labs/ remain the repository source of truth.
package lpicdaily

import "embed"

// BuiltinFS contains the built-in LPIC curriculum, content contracts and
// authored labs used by the shipped application.
//
//go:embed curriculum/lpic-1-v5/*.json schemas/*.schema.json labs/lpic-1-v5/*/*/*.json labs/lpic-1-v5/*/*/*.sh labs/lpic-1-v5/*/*/hints/*.json
var BuiltinFS embed.FS
