#!/usr/bin/env bash
set -euo pipefail
for tool in gcc wall systemctl; do command -v "$tool" >/dev/null; done
install -d -m 0755 /var/lib/lpic-wall /run/lpic-wall /usr/local/libexec
test -e /run/utmp || install -m 0664 /dev/null /run/utmp
cat > /tmp/lpic-wall-receiver.c <<'EOF'
#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <pty.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <sys/utmpx.h>
#include <sys/time.h>
#include <unistd.h>
static volatile sig_atomic_t running = 1;
static void stop(int sig) { (void)sig; running = 0; }
int main(int argc, char **argv) {
    if (argc != 2 || (strcmp(argv[1], "a") && strcmp(argv[1], "b"))) return 64;
    const char *name = argv[1];
    int master = -1, slave = -1;
    char tty[128], dest[256], logpath[256];
    if (openpty(&master, &slave, tty, NULL, NULL) != 0) return 1;
    if (strncmp(tty, "/dev/pts/", 9) != 0) return 2;
    snprintf(dest, sizeof dest, "/run/lpic-wall/tty-%s", name);
    snprintf(logpath, sizeof logpath, "/var/lib/lpic-wall/received-%s.log", name);
    FILE *line = fopen(dest, "w");
    if (!line) return 3;
    fprintf(line, "%s\n", tty);
    fclose(line);
    int log = open(logpath, O_CREAT | O_TRUNC | O_WRONLY, 0644);
    if (log < 0) return 4;
    struct utmpx entry;
    memset(&entry, 0, sizeof entry);
    entry.ut_type = USER_PROCESS;
    entry.ut_pid = getpid();
    snprintf(entry.ut_id, sizeof entry.ut_id, "lw%s", name);
    snprintf(entry.ut_line, sizeof entry.ut_line, "%s", tty + 5);
    snprintf(entry.ut_user, sizeof entry.ut_user, "lpic-operator");
    gettimeofday((struct timeval *)&entry.ut_tv, NULL);
    setutxent();
    if (!pututxline(&entry)) return 5;
    endutxent();
    signal(SIGTERM, stop);
    char buf[1024];
    while (running) {
        ssize_t n = read(master, buf, sizeof buf);
        if (n > 0) {
            if (write(log, buf, (size_t)n) != n) return 6;
            fsync(log);
        } else if (n < 0 && errno == EINTR) {
            continue;
        } else {
            break;
        }
    }
    entry.ut_type = DEAD_PROCESS;
    memset(entry.ut_user, 0, sizeof entry.ut_user);
    setutxent();
    pututxline(&entry);
    endutxent();
    close(log); close(master); close(slave);
    return 0;
}
EOF
gcc -O2 -Wall -Wextra -o /usr/local/libexec/lpic-wall-receiver /tmp/lpic-wall-receiver.c -lutil
cat > /etc/systemd/system/lpic-wall-observer@.service <<'EOF'
[Unit]
Description=LPIC maintenance recipient PTY %i

[Service]
Type=simple
ExecStart=/usr/local/libexec/lpic-wall-receiver %i
Restart=no
EOF
systemctl daemon-reload
systemctl start lpic-wall-observer@a.service lpic-wall-observer@b.service
for n in a b; do
  for i in $(seq 1 30); do
    test -s "/run/lpic-wall/tty-$n" && break
    sleep 0.1
  done
  test -s "/run/lpic-wall/tty-$n"
  systemctl is-active --quiet "lpic-wall-observer@$n.service"
  test ! -s "/var/lib/lpic-wall/received-$n.log"
done
test "$(cat /run/lpic-wall/tty-a)" != "$(cat /run/lpic-wall/tty-b)"
