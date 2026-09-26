# Fedora Phase-1 lab image

Local development tag expected by the first lab:

```text
localhost/lpic-daily/fedora-phase1:1
```

The application never pulls this image automatically. The trusted install/development workflow builds or imports it before labs run.

Development build:

```bash
podman build -t localhost/lpic-daily/fedora-phase1:1 labs/images/fedora-phase1
```

The current Containerfile uses the Fedora 44 release tag only as a development input. A release pipeline must resolve and verify a digest, record provenance, and build/publish an immutable LPIC Daily image identity before this image can be treated as a release artifact.
