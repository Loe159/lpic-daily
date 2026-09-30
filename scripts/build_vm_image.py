#!/usr/bin/env python3
"""Build and install one trusted LPIC Daily VM image.

This is an explicit maintainer/admin workflow. The application never invokes it.
"""
from pathlib import Path
from urllib.request import Request, urlopen
import argparse
import base64
import datetime as dt
import hashlib
import json
import os
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
SOURCES = ROOT / "packaging" / "vm-images" / "sources.json"
CATALOG_SCHEMA_VERSION = "1.0.0"

def run(*argv):
    subprocess.run(argv, check=True)

def output(*argv):
    return subprocess.check_output(argv, text=True).strip()

def require_tools(*names):
    missing = [name for name in names if shutil.which(name) is None]
    if missing:
        raise SystemExit("missing required tool(s): " + ", ".join(missing))

def ensure_qemu_traversable_directory(path):
    path.mkdir(parents=True, exist_ok=True)
    mode = path.stat().st_mode & 0o777
    os.chmod(path, mode | 0o011)

def download(url, destination):
    request = Request(url, headers={"User-Agent": "LPIC-Daily-image-builder/1"})
    with urlopen(request, timeout=120) as response, destination.open("wb") as target:
        shutil.copyfileobj(response, target)

def digest(path, algorithm):
    h = hashlib.new(algorithm)
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            h.update(block)
    return h.digest()

def expected_digest(image):
    integrity = image["integrity"]
    value = integrity.get("value")
    if not isinstance(value, str) or not value:
        raise SystemExit(f"{image['id']}: integrity.value must be pinned in sources.json")
    encoding = integrity["encoding"]
    return bytes.fromhex(value) if encoding == "hex" else base64.b64decode(value, validate=True)

def recipe_commands(image):
    common = [
        "systemctl enable qemu-guest-agent.service",
        "systemctl enable serial-getty@ttyS0.service",
        "truncate -s 0 /etc/machine-id",
        "rm -f /var/lib/dbus/machine-id /etc/ssh/ssh_host_*",
    ]
    recipe = image["recipe"]
    distribution = image["distribution"]
    if recipe == "fedora-cloud-v1":
        expected_distribution = "fedora"
        commands = [
            "grubby --update-kernel=ALL --args='console=tty0 console=ttyS0,115200n8'",
            "sed -i '/^GRUB_TERMINAL_INPUT=/d; /^GRUB_TERMINAL_OUTPUT=/d; /^GRUB_SERIAL_COMMAND=/d; /^GRUB_TIMEOUT_STYLE=/d; /^GRUB_TIMEOUT=/d' /etc/default/grub",
            """printf '%s\n' 'GRUB_TERMINAL_INPUT="console serial"' 'GRUB_TERMINAL_OUTPUT="console serial"' 'GRUB_SERIAL_COMMAND="serial --unit=0 --speed=115200 --word=8 --parity=no --stop=1"' 'GRUB_TIMEOUT_STYLE="menu"' 'GRUB_TIMEOUT="5"' >> /etc/default/grub""",
            "grub2-mkconfig -o /boot/grub2/grub.cfg",
        ]
    elif recipe == "debian-cloud-v1":
        expected_distribution = "debian"
        commands = [
            "grep -q 'console=ttyS0,115200n8' /etc/default/grub || sed -i 's/^GRUB_CMDLINE_LINUX="/GRUB_CMDLINE_LINUX="console=tty0 console=ttyS0,115200n8 /' /etc/default/grub",
            "update-grub",
        ]
    elif recipe == "opensuse-cloud-v1":
        expected_distribution = "opensuse"
        commands = [
            "grep -q 'console=ttyS0,115200n8' /etc/default/grub || sed -i 's/^GRUB_CMDLINE_LINUX_DEFAULT="/GRUB_CMDLINE_LINUX_DEFAULT="console=tty0 console=ttyS0,115200n8 /' /etc/default/grub",
            "grub2-mkconfig -o /boot/grub2/grub.cfg",
        ]
    else:
        raise SystemExit(f"unsupported build recipe {recipe!r}")
    if distribution != expected_distribution:
        raise SystemExit(
            f"{image['id']}: recipe {recipe!r} requires distribution "
            f"{expected_distribution!r}, got {distribution!r}"
        )
    return commands + common

def qemu_virtual_size_mb(path):
    info = json.loads(output("qemu-img", "info", "--output=json", str(path)))
    if info.get("format") != "qcow2":
        raise SystemExit(f"{path}: expected qcow2, got {info.get('format')!r}")
    return (int(info["virtual-size"]) + (1024 * 1024 - 1)) // (1024 * 1024)

def atomic_write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, tmp = tempfile.mkstemp(prefix=path.name + ".", dir=path.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            json.dump(value, stream, indent=2, sort_keys=True)
            stream.write("\n")
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(tmp, path)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("image_id")
    parser.add_argument("--image-root", type=Path, required=True)
    parser.add_argument("--cache-dir", type=Path)
    args = parser.parse_args()

    require_tools("qemu-img", "virt-customize")
    manifest = json.loads(SOURCES.read_text(encoding="utf-8"))
    try:
        image = next(item for item in manifest["images"] if item["id"] == args.image_id)
    except StopIteration:
        raise SystemExit(f"unknown image id {args.image_id!r}")

    image_root = args.image_root.expanduser().resolve()
    ensure_qemu_traversable_directory(image_root)
    cache_dir = (args.cache_dir or (image_root / ".source-cache")).expanduser().resolve()
    cache_dir.mkdir(parents=True, exist_ok=True)

    source_path = cache_dir / image["source_filename"]
    if not source_path.exists():
        print(f"downloading {image['source_url']}")
        download(image["source_url"], source_path)

    got = digest(source_path, image["integrity"]["algorithm"])
    want = expected_digest(image)
    if got != want:
        raise SystemExit(f"source integrity mismatch for {image['id']}")
    print(f"verified source integrity: {image['integrity']['algorithm']}")

    target_dir = image_root / image["distribution"]
    ensure_qemu_traversable_directory(target_dir)
    final_path = target_dir / (image["id"] + ".qcow2")

    # Keep the build workspace on the image-root filesystem so the final
    # os.replace() remains atomic even when /tmp is a separate mount.
    with tempfile.TemporaryDirectory(prefix=".lpic-daily-vm-build-", dir=image_root) as tmp:
        work = Path(tmp) / "work.qcow2"
        run("qemu-img", "convert", "-f", "qcow2", "-O", "qcow2", str(source_path), str(work))
        current_mb = qemu_virtual_size_mb(work)
        target_mb = image["virtual_size_mb"]
        if current_mb > target_mb:
            raise SystemExit(
                f"{image['id']}: source virtual size {current_mb} MiB exceeds "
                f"declared target {target_mb} MiB"
            )
        if current_mb < target_mb:
            run("qemu-img", "resize", str(work), f"{target_mb}M")

        command = [
            "virt-customize", "-a", str(work), "--network",
            "--install", ",".join(image["packages"]),
        ]
        for item in recipe_commands(image):
            command += ["--run-command", item]
        run(*command)
        run("qemu-img", "check", str(work))

        # Re-convert to a clean standalone qcow2, verify the artifact while it
        # is still temporary, then install it atomically. A failed contract
        # check must never replace the previously trusted image.
        installed_tmp = Path(tmp) / "installed.qcow2"
        run("qemu-img", "convert", "-f", "qcow2", "-O", "qcow2", "-c", str(work), str(installed_tmp))
        virtual_size = qemu_virtual_size_mb(installed_tmp)
        if virtual_size != target_mb:
            raise SystemExit(
                f"{image['id']}: built virtual size {virtual_size} MiB does not match "
                f"declared target {target_mb} MiB"
            )
        final_sha = digest(installed_tmp, "sha256").hex()
        os.chmod(installed_tmp, 0o444)
        os.replace(installed_tmp, final_path)
    relative_path = str(final_path.relative_to(image_root))
    built_at = dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")
    entry = {
        "id": image["id"],
        "relative_path": relative_path,
        "sha256": final_sha,
        "format": "qcow2",
        "architecture": image["architecture"],
        "distribution": image["distribution"],
        "version": image["version"],
        "virtual_size_mb": virtual_size,
        "firmware": image["firmware"],
        "provenance": {
            "source_url": image["source_url"],
            "source_integrity": {
                "algorithm": image["integrity"]["algorithm"],
                "encoding": image["integrity"]["encoding"],
                "value": image["integrity"]["value"],
            },
            "build_recipe": image["recipe"],
            "built_at": built_at,
        },
    }
    catalog_path = image_root / "catalog.json"
    if catalog_path.exists():
        catalog = json.loads(catalog_path.read_text(encoding="utf-8"))
        if catalog.get("schema_version") != CATALOG_SCHEMA_VERSION:
            raise SystemExit("refusing to modify unsupported catalog schema")
        entries = [item for item in catalog.get("images", []) if item.get("id") != image["id"]]
    else:
        entries = []
    entries.append(entry)
    entries.sort(key=lambda item: item["id"])
    atomic_write_json(catalog_path, {"schema_version": CATALOG_SCHEMA_VERSION, "images": entries})
    print(f"installed immutable image: {final_path}")
    print(f"catalog updated: {catalog_path}")
    print(f"sha256={final_sha}")

if __name__ == "__main__":
    main()
