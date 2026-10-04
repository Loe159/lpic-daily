// Package lpicdaily exposes immutable built-in project assets.
//
// Application packages should treat BuiltinFS as read-only. The files under
// curriculum/, schemas/ and labs/ remain the repository source of truth.
package lpicdaily

import "embed"

// BuiltinFS contains the built-in LPIC curriculum, content contracts and
// authored labs used by the shipped application.
//
//go:embed curriculum/lpic-1-v5/*.json schemas/*.schema.json labs/lpic-1-v5/*/*/*.json labs/lpic-1-v5/*/*/*.sh labs/lpic-1-v5/*/*/hints/*.json content/lpic-1-v5/lessons content/lpic-1-v5/questions packaging/systemd/*.service packaging/systemd/*.timer packaging/desktop/*.desktop labs/images/fedora-phase1/* scripts/provision_vm_storage.sh scripts/build_vm_image.py packaging/vm-images/sources.json
var BuiltinFS embed.FS
