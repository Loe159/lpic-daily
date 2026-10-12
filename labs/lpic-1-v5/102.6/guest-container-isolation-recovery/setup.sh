#!/usr/bin/env bash
set -euo pipefail
for cmd in gcc podman tar; do command -v "$cmd" >/dev/null; done
test -d /run/systemd/system
install -d -m 0755 /var/lib/lpic-container /tmp/lpic-container-root
printf 'archive-queue=preserve-54\n' > /var/lib/lpic-container/queue.txt
cat > /tmp/lpic-probe.c <<'EOF'
#include <stdio.h>
#include <string.h>
#include <sys/utsname.h>
#include <unistd.h>
int main(int argc, char **argv) {
    if (argc == 2 && strcmp(argv[1], "--kernel") == 0) {
        struct utsname u;
        if (uname(&u) != 0) return 1;
        puts(u.release);
        return 0;
    }
    if (argc == 2 && strcmp(argv[1], "--pid") == 0) {
        printf("%ld\n", (long)getpid());
        return 0;
    }
    for (;;) sleep(30);
}
EOF
gcc -O2 -static -o /tmp/lpic-container-root/probe /tmp/lpic-probe.c
tar -C /tmp/lpic-container-root -cf /tmp/lpic-container-image.tar probe
podman import /tmp/lpic-container-image.tar localhost/lpic-guest-probe:1 >/dev/null
podman run --pull=never -d --name lpic-worker --network=host --pid=host localhost/lpic-guest-probe:1 /probe >/dev/null
test "$(podman inspect --format '{{.State.Running}}' lpic-worker)" = true
test "$(podman inspect --format '{{.HostConfig.NetworkMode}}' lpic-worker)" = host
