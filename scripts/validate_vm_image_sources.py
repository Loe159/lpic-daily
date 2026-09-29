#!/usr/bin/env python3
from pathlib import Path
from urllib.parse import urlparse
import base64
import json
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "packaging" / "vm-images" / "sources.json"
ID_RE = re.compile(r"^[a-z0-9][a-z0-9._-]{2,63}$")
HEX_RE = re.compile(r"^[0-9a-f]+$")
ALLOWED_DISTRIBUTIONS = {"fedora", "debian", "opensuse"}
ALLOWED_FIRMWARE = {"bios", "uefi"}
ALLOWED_ALGORITHMS = {"sha256": 32, "sha512": 64}
ALLOWED_ENCODINGS = {"hex", "base64"}

def fail(message):
    print("VM image source validation FAILED:", message)
    raise SystemExit(1)

def https_url(value, field):
    if not isinstance(value, str) or urlparse(value).scheme != "https":
        fail(f"{field} must be an https URL")
    lowered = value.lower()
    for floating in ("/latest/", "/current/", "/daily/"):
        if floating in lowered:
            fail(f"{field} must be version-pinned, not {floating.strip('/')}")

def decoded_integrity(integrity, image_id):
    algorithm = integrity.get("algorithm")
    encoding = integrity.get("encoding")
    if algorithm not in ALLOWED_ALGORITHMS:
        fail(f"{image_id}: unsupported integrity algorithm {algorithm!r}")
    if encoding not in ALLOWED_ENCODINGS:
        fail(f"{image_id}: unsupported integrity encoding {encoding!r}")
    value = integrity.get("value")
    if integrity.get("checksum_url"):
        fail(f"{image_id}: checksum_url is not accepted; pin the digest value in the repository")
    if not isinstance(value, str) or not value:
        fail(f"{image_id}: integrity.value is required")
    try:
        raw = bytes.fromhex(value) if encoding == "hex" else base64.b64decode(value, validate=True)
    except (ValueError, TypeError) as exc:
        fail(f"{image_id}: invalid {encoding} checksum: {exc}")
    if len(raw) != ALLOWED_ALGORITHMS[algorithm]:
        fail(f"{image_id}: wrong {algorithm} digest length")

def main():
    try:
        manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
    except Exception as exc:
        fail(f"cannot read {MANIFEST.relative_to(ROOT)}: {exc}")
    if manifest.get("schema_version") != "1.0.0":
        fail("unsupported schema_version")
    images = manifest.get("images")
    if not isinstance(images, list) or len(images) < 3:
        fail("Fedora, Debian and openSUSE image definitions are required")

    seen = set()
    distributions = set()
    for image in images:
        image_id = image.get("id")
        if not isinstance(image_id, str) or not ID_RE.fullmatch(image_id):
            fail(f"invalid image id {image_id!r}")
        if image_id in seen:
            fail(f"duplicate image id {image_id}")
        seen.add(image_id)
        distribution = image.get("distribution")
        if distribution not in ALLOWED_DISTRIBUTIONS:
            fail(f"{image_id}: unsupported distribution {distribution!r}")
        distributions.add(distribution)
        if image.get("architecture") != "x86_64":
            fail(f"{image_id}: only x86_64 is supported in Phase 2")
        if not isinstance(image.get("version"), str) or not image["version"].strip():
            fail(f"{image_id}: version is required")
        https_url(image.get("source_url"), f"{image_id}.source_url")
        filename = image.get("source_filename")
        if not isinstance(filename, str) or "/" in filename or not filename.endswith(".qcow2"):
            fail(f"{image_id}: source_filename must be a qcow2 basename")
        decoded_integrity(image.get("integrity", {}), image_id)
        size = image.get("virtual_size_mb")
        if not isinstance(size, int) or not 1024 <= size <= 32768:
            fail(f"{image_id}: virtual_size_mb must be 1024..32768")
        firmware = image.get("firmware")
        if not isinstance(firmware, list) or not firmware or set(firmware) - ALLOWED_FIRMWARE:
            fail(f"{image_id}: invalid firmware list")
        packages = image.get("packages")
        if not isinstance(packages, list) or not packages or any(not isinstance(x, str) or not x for x in packages):
            fail(f"{image_id}: non-empty package list required")
        if "qemu-guest-agent" not in packages:
            fail(f"{image_id}: qemu-guest-agent is required")
        if not isinstance(image.get("recipe"), str) or not image["recipe"]:
            fail(f"{image_id}: recipe is required")

    if distributions != ALLOWED_DISTRIBUTIONS:
        fail(f"required distributions are {sorted(ALLOWED_DISTRIBUTIONS)}, got {sorted(distributions)}")
    print(f"VM image source validation OK: {len(images)} pinned definitions")

if __name__ == "__main__":
    main()
