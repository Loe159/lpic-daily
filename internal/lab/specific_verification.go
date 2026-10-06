package lab

import (
	"fmt"

	"github.com/Loe159/lpic-daily/internal/curriculum"
)

type inspectionProbePair struct {
	diagnostic string
	transfer   string
}

func deterministicSystemControlExercise(concept curriculum.Concept, root string, variant int) (string, []CheckDefinition, string) {
	check := func(script string) []CheckDefinition {
		return []CheckDefinition{deterministicCommandCheck(concept.ID, script)}
	}
	switch concept.PedagogyOrder {
	case 3:
		target := "multi-user.target"
		if variant == 1 {
			target = "graphical.target"
		}
		return fmt.Sprintf("Change le target systemd par défaut vers %s avec systemctl set-default. Ne redémarre pas la VM.", target),
			check(fmt.Sprintf("test \"$(systemctl get-default)\" = %s", target)), ""
	case 5:
		token := "LPIC-DIAGNOSTIC-WALL"
		if variant == 1 {
			token = "LPIC-TRANSFER-WALL"
		}
		scriptPath := root + "/wall-message.sh"
		verify := fmt.Sprintf("set -eu; test -x %q; grep -Eq '^wall[[:space:]]+.*%s' %q; %q", scriptPath, token, scriptPath, scriptPath)
		return fmt.Sprintf("Crée le script exécutable %s contenant une commande wall qui diffuse exactement le marqueur %s. La validation exécute le script.", scriptPath, token), check(verify), ""
	default:
		suffix := "diagnostic"
		if variant == 1 {
			suffix = "transfer"
		}
		unitName := "lpic-termination-" + suffix + ".service"
		unitPath := "/etc/systemd/system/" + unitName
		marker := root + "/termination-" + suffix + ".started"
		pidPath := root + "/termination-" + suffix + ".pid"
		verify := fmt.Sprintf("set -eu; test -f %q; test -s %q; test -s %q; ! kill -0 \"$(cat %q)\" 2>/dev/null; ! systemctl is-active --quiet %s; stamp=$(systemctl show -p InactiveEnterTimestampMonotonic --value %s 2>/dev/null || true); test -n \"$stamp\"; test \"$stamp\" != 0", unitPath, marker, pidPath, pidPath, unitName, unitName)
		return fmt.Sprintf("Crée %s. Son ExecStart doit écrire son PID dans %s, créer %s puis rester actif. Recharge systemd, démarre l’unité, vérifie qu’elle est active puis arrête-la proprement avec systemctl stop.", unitPath, pidPath, marker), check(verify), ""
	}
}

func deterministicGRUBGenerationExercise(concept curriculum.Concept, root string, variant int) (string, []CheckDefinition, string) {
	output := root + "/generated-grub.cfg"
	if variant == 0 {
		scriptPath := root + "/generate-grub.sh"
		verify := fmt.Sprintf("set -eu; test -x %q; grep -Eq 'grub2?-mkconfig' %q; rm -f %q; %q; test -s %q; grep -Eq '(insmod|menuentry|blscfg|linux)' %q", scriptPath, scriptPath, output, scriptPath, output, output)
		return fmt.Sprintf("Crée le script exécutable %s qui génère la configuration GRUB actuelle dans %s avec grub2-mkconfig ou grub-mkconfig. La validation régénère le fichier.", scriptPath, output), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, ""
	}
	fragment := "/etc/grub.d/41_lpic_daily"
	scriptPath := root + "/generate-grub-transfer.sh"
	output = root + "/generated-grub-transfer.cfg"
	verify := fmt.Sprintf("set -eu; test -x %q; grep -Fq 'LPIC Daily Transfer' %q; test -x %q; grep -Eq 'grub2?-mkconfig' %q; rm -f %q; %q; test -s %q; grep -Fq 'LPIC Daily Transfer' %q", fragment, fragment, scriptPath, scriptPath, output, scriptPath, output, output)
	return fmt.Sprintf("Crée le fragment exécutable %s qui émet une entrée menuentry nommée LPIC Daily Transfer, puis crée %s pour régénérer GRUB dans %s. La validation régénère et vérifie l’entrée.", fragment, scriptPath, output), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, ""
}

func specificInspectionProbes(objectiveID string, order int) (string, string, bool) {
	probes := map[string][]inspectionProbePair{
		"101.1": {
			{"for p in /proc /sys /dev; do stat -Lc '%n|%F|%a' \"$p\"; done", "grep -m1 -E 'model name|Hardware' /proc/cpuinfo 2>/dev/null || true; find /sys/class -mindepth 1 -maxdepth 1 -printf '%f\\n' 2>/dev/null | sort | head -16; ls -l /dev/null /dev/zero"},
			{"{ lspci -nn 2>/dev/null || true; lsusb 2>/dev/null || true; } | head -24", "grep -m1 -E 'model name|Hardware' /proc/cpuinfo 2>/dev/null || true; lsblk -ndo NAME,TYPE,MODEL,TRAN 2>/dev/null | head -16"},
			{"lsmod 2>/dev/null | sed -n '1,16p'", "m=$(lsmod 2>/dev/null | awk 'NR==2{print $1}'); if test -n \"$m\"; then modinfo \"$m\" 2>/dev/null | sed -n '1,16p'; else echo no-loaded-module; fi"},
			{"printf 'pci\\n'; lspci -nn 2>/dev/null | head -12; printf 'usb\\n'; lsusb 2>/dev/null | head -12", "printf 'pci-sysfs\\n'; find /sys/bus/pci/devices -mindepth 1 -maxdepth 1 -printf '%f\\n' 2>/dev/null | sort | head -12; printf 'usb-sysfs\\n'; find /sys/bus/usb/devices -mindepth 1 -maxdepth 1 -printf '%f\\n' 2>/dev/null | sort | head -12"},
			{"{ udevadm info --export-db 2>/dev/null || true; } | sed -n '1,20p'; find /sys/class -mindepth 1 -maxdepth 1 -printf '%f\\n' 2>/dev/null | sort | head -12", "{ busctl --system list 2>/dev/null || true; } | sed -n '1,16p'; udevadm control --version 2>/dev/null || true"},
			{"lsblk -o NAME,TYPE,TRAN,ROTA,SIZE 2>/dev/null", "for d in /sys/block/*; do printf '%s|' \"$(basename \"$d\")\"; cat \"$d/queue/rotational\" 2>/dev/null || echo ?; done | head -20"},
			{"lspci -k 2>/dev/null | sed -n '1,28p'", "m=$(lsmod 2>/dev/null | awk 'NR==2{print $1}'); if test -n \"$m\"; then printf 'dry-run|%s\\n' \"$m\"; modprobe -n \"$m\" 2>/dev/null || true; fi; find /sys/bus/pci/drivers -mindepth 1 -maxdepth 1 -type d -printf '%f\\n' 2>/dev/null | sort | head -12"},
			{"{ lsusb -t 2>/dev/null || lsusb 2>/dev/null || true; }", "find /sys/bus/usb/devices -mindepth 1 -maxdepth 1 -printf '%f\\n' 2>/dev/null | sort | head -20; { udevadm info --export-db 2>/dev/null || true; } | grep -m5 -E 'usb|USB' || true"},
		},
		"101.2": {
			{"if test -d /sys/firmware/efi; then echo firmware=UEFI; else echo firmware=BIOS; fi; printf 'cmdline|'; cat /proc/cmdline; printf 'pid1|'; cat /proc/1/comm", "find /boot -maxdepth 2 -type f -printf '%f\\n' 2>/dev/null | sort | head -20; ls -1 /boot/*initramfs* /boot/*initrd* 2>/dev/null | head -4 || true"},
			{"if test -d /sys/firmware/efi; then echo UEFI; else echo BIOS; fi", "if test -d /sys/firmware/efi; then find /sys/firmware/efi -maxdepth 2 -type d -printf '%P\\n' 2>/dev/null | sort | head -16; else echo legacy-firmware-interface; fi"},
			{"cat /proc/cmdline", "tr ' ' '\\n' < /proc/cmdline | sed '/^$/d' | sort"},
			{"ls -lh /boot/*initramfs* /boot/*initrd* 2>/dev/null | head -6 || true", "f=$(ls -1 /boot/*initramfs* /boot/*initrd* 2>/dev/null | head -1); if test -n \"$f\" && command -v lsinitrd >/dev/null 2>&1; then lsinitrd -m \"$f\" 2>/dev/null | head -20; else echo initramfs-tool-unavailable; fi"},
			{"printf 'pid1|'; cat /proc/1/comm; systemctl --version 2>/dev/null | head -2 || true", "test -e /etc/inittab && sed -n '1,12p' /etc/inittab || echo no-inittab; find /etc/systemd/system -maxdepth 1 -type l -printf '%f -> %l\\n' 2>/dev/null | sort | head -12"},
			{"{ journalctl -b -p warning..alert -n 16 --no-pager 2>/dev/null || true; }; { dmesg --level=warn,err 2>/dev/null | tail -12 || true; }", "{ journalctl --list-boots --no-pager 2>/dev/null || true; } | tail -8; systemd-analyze time 2>/dev/null || true"},
			{"for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg; do test -f \"$p\" && grep -E '^[[:space:]]*(menuentry|linux|linuxefi|initrd|initrdefi|set root)' \"$p\" | head -20; done", "{ grub2-editenv list 2>/dev/null || grub-editenv list 2>/dev/null || true; }; for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg; do test -f \"$p\" && grep -E '^[[:space:]]*(search|set root)' \"$p\" | head -12; done"},
		},
		"101.3": {
			{"systemctl get-default 2>/dev/null || true; systemctl list-unit-files --type=target --no-pager 2>/dev/null | sed -n '1,16p'", "readlink -f /etc/systemd/system/default.target 2>/dev/null || true; test -e /etc/inittab && sed -n '1,12p' /etc/inittab || true"},
			{"systemctl list-dependencies rescue.target --plain --no-pager 2>/dev/null | sed -n '1,20p'", "{ systemctl cat rescue.target 2>/dev/null || true; systemctl cat emergency.target 2>/dev/null || true; } | sed -n '1,28p'"},
			{"systemctl get-default 2>/dev/null || true", "readlink -f /etc/systemd/system/default.target 2>/dev/null || true"},
			{"shutdown --help 2>/dev/null | sed -n '1,16p' || true", "{ systemctl cat systemd-poweroff.service 2>/dev/null || true; systemctl cat systemd-reboot.service 2>/dev/null || true; } | sed -n '1,28p'"},
			{"who 2>/dev/null || true; command -v wall 2>/dev/null || true", "loginctl list-sessions --no-legend 2>/dev/null || true; wall --help 2>/dev/null | head -10 || true"},
			{"systemctl list-dependencies \"$(systemctl get-default 2>/dev/null)\" --plain --no-pager 2>/dev/null | sed -n '1,24p'", "systemctl show \"$(systemctl get-default 2>/dev/null)\" -p Wants -p Requires -p After 2>/dev/null | sed -n '1,16p'"},
			{"systemctl status acpid --no-pager 2>/dev/null | sed -n '1,16p' || true", "find /sys/firmware/acpi/tables -maxdepth 1 -type f -printf '%f\\n' 2>/dev/null | sort | head -16; command -v acpid 2>/dev/null || true"},
			{"systemctl show -p DefaultTimeoutStopSec 2>/dev/null || true", "systemctl cat systemd-poweroff.service 2>/dev/null | sed -n '1,20p' || true"},
		},
		"102.2": {
			{"printf 'legacy|'; for p in /boot/grub/menu.lst /boot/grub/grub.conf; do test -e \"$p\" && echo \"$p\"; done; printf 'grub2|'; for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg; do test -e \"$p\" && echo \"$p\"; done", "for p in /boot/grub/menu.lst /boot/grub/grub.conf /boot/grub2/grub.cfg /boot/grub/grub.cfg; do test -e \"$p\" && printf '%s|' \"$p\" && head -1 \"$p\"; done"},
			{"for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg /boot/grub/grub.conf; do test -e \"$p\" && echo \"$p\"; done; command -v grub2-install 2>/dev/null || command -v grub-install 2>/dev/null || true", "command -v grub2-mkconfig 2>/dev/null || command -v grub-mkconfig 2>/dev/null || true; find /etc/default /etc/grub.d -maxdepth 1 -type f -printf '%p\\n' 2>/dev/null | sort | head -16"},
			{"for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg; do test -f \"$p\" && grep -Ei 'menuentry|submenu|recovery|rescue' \"$p\" | head -16; done", "{ grub2-editenv list 2>/dev/null || grub-editenv list 2>/dev/null || true; }; find /boot -maxdepth 2 -type f -iname '*rescue*' -printf '%p\\n' 2>/dev/null | head -8"},
			{"dd if=/dev/vda bs=512 count=1 status=none 2>/dev/null | sha256sum | awk '{print $1}'", "{ fdisk -l /dev/vda 2>/dev/null || parted -sm /dev/vda print 2>/dev/null || true; } | sed -n '1,20p'"},
			{"command -v grub2-mkconfig 2>/dev/null || command -v grub-mkconfig 2>/dev/null || true", "for p in /etc/default/grub /etc/grub.d; do test -e \"$p\" && stat -Lc '%n|%F|%a' \"$p\"; done"},
			{"for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg; do test -f \"$p\" && grep -E '^[[:space:]]*(menuentry|submenu|set root|search)' \"$p\" | head -18; done", "{ grub2-editenv list 2>/dev/null || grub-editenv list 2>/dev/null || true; }; for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg; do test -f \"$p\" && grep -E '^[[:space:]]*(linux|linuxefi|initrd|initrdefi)' \"$p\" | head -12; done"},
		},
		"102.3": {
			{"ldd /bin/sh 2>/dev/null | sed -n '1,14p'", "ldd /bin/ls 2>/dev/null | sed -n '1,14p'"},
			{"{ grep -hv '^[[:space:]]*#' /etc/ld.so.conf /etc/ld.so.conf.d/*.conf 2>/dev/null || true; } | sed '/^[[:space:]]*$/d' | sort", "printf 'LD_LIBRARY_PATH|%s\\n' \"${LD_LIBRARY_PATH-}\"; for p in /lib /usr/lib /lib64 /usr/lib64; do test -e \"$p\" && readlink -f \"$p\"; done"},
			{"ldconfig -p 2>/dev/null | sed -n '1,18p'", "stat -Lc '%n|%s|%Y' /etc/ld.so.cache 2>/dev/null || true; strings /etc/ld.so.cache 2>/dev/null | head -12 || true"},
			{"printf '%s\\n' \"${LD_LIBRARY_PATH-}\"", "env LD_LIBRARY_PATH=/tmp ldd /bin/sh 2>/dev/null | sed -n '1,10p'"},
			{"ldd /bin/sh 2>/dev/null | sed -n '1,14p'", "ldd /bin/ls 2>/dev/null | awk '/not found|=>/{print}' | head -14"},
		},
		"102.4": {
			{"command -v dpkg; command -v apt; dpkg --version | head -1; apt --version | head -1", "stat -Lc '%n|%F' /var/lib/dpkg /var/lib/apt 2>/dev/null"},
			{"dpkg-query -W bash coreutils 2>/dev/null", "apt-cache policy bash 2>/dev/null | sed -n '1,14p'; dpkg -s bash 2>/dev/null | grep -E '^(Status|Version):'"},
			{"grep -RhEv '^[[:space:]]*(#|$)' /etc/apt/sources.list /etc/apt/sources.list.d 2>/dev/null | head -16 || true", "apt-cache policy 2>/dev/null | sed -n '1,22p'"},
			{"apt-cache depends bash 2>/dev/null | sed -n '1,18p'", "apt-cache rdepends bash 2>/dev/null | sed -n '1,18p'"},
			{"dpkg -L bash 2>/dev/null | sed -n '1,18p'", "dpkg-query -W bash 2>/dev/null; dpkg-query -s bash 2>/dev/null | grep -E '^(Status|Conffiles):' | head -12"},
			{"command -v dpkg-reconfigure 2>/dev/null || true; ls -1 /var/cache/debconf 2>/dev/null | head -12 || true", "dpkg-query -W debconf 2>/dev/null || true; find /var/lib/dpkg/info -maxdepth 1 -name 'bash.*' -printf '%f\\n' 2>/dev/null | sort | head -12"},
			{"dpkg -S /bin/sh 2>/dev/null || true", "target=$(readlink -f /bin/sh); printf 'target|%s\\n' \"$target\"; dpkg -S \"$target\" 2>/dev/null || true"},
		},
		"102.5": {
			{"rpm --version; rpm --eval '%{_dbpath}'", "rpm -qa 2>/dev/null | sort | head -16"},
			{"rpm -q bash 2>/dev/null || true; rpm -ql bash 2>/dev/null | head -12 || true", "rpm -qi bash 2>/dev/null | sed -n '1,14p' || true; rpm -qf /bin/sh 2>/dev/null || true"},
			{"rpm -V bash 2>/dev/null | head -12 || true", "rpm -Va 2>/dev/null | head -12 || true"},
			{"{ dnf repoquery --requires bash 2>/dev/null || yum deplist bash 2>/dev/null || true; } | head -18", "{ zypper info bash 2>/dev/null || zypper search -s bash 2>/dev/null || true; } | head -18"},
			{"{ dnf repolist 2>/dev/null || yum repolist 2>/dev/null || true; } | sed -n '1,16p'; find /etc/yum.repos.d -maxdepth 1 -type f -printf '%f\\n' 2>/dev/null | sort | head -12", "{ dnf repoquery bash 2>/dev/null || yum list bash 2>/dev/null || true; } | head -16"},
			{"command -v zypper 2>/dev/null || true; zypper lr 2>/dev/null | sed -n '1,16p' || true", "zypper search -s bash 2>/dev/null | sed -n '1,16p' || true; rpm -q bash 2>/dev/null || true"},
			{"rpm -qf /bin/sh 2>/dev/null || true", "target=$(readlink -f /bin/sh); printf 'target|%s\\n' \"$target\"; rpm -qf \"$target\" 2>/dev/null || true"},
		},
		"102.6": {
			{"systemd-detect-virt 2>/dev/null || true; cat /proc/1/cgroup 2>/dev/null | head -12", "lsns -t pid 2>/dev/null | head -12 || true; test -f /.dockerenv && echo docker-env || true"},
			{"lsblk -ndo NAME,TYPE,SIZE 2>/dev/null; ip -o link show 2>/dev/null | sed -n '1,10p'", "find /sys/class/net -mindepth 1 -maxdepth 1 -printf '%f\\n' 2>/dev/null | sort; ls /dev/vd* /dev/xvd* 2>/dev/null | head -12 || true"},
			{"cloud-init status 2>/dev/null || true; find /var/lib/cloud -maxdepth 2 -type d -printf '%P\\n' 2>/dev/null | sort | head -14", "printf 'machine-id|'; cat /etc/machine-id 2>/dev/null; hostname"},
			{"printf 'machine-id|'; cat /etc/machine-id 2>/dev/null; sha256sum /etc/ssh/ssh_host_*_key.pub 2>/dev/null | sort | head -12", "find /etc/ssh -maxdepth 1 -name 'ssh_host_*_key.pub' -printf '%f\\n' 2>/dev/null | sort; command -v dbus-uuidgen 2>/dev/null || true"},
			{"lsmod 2>/dev/null | grep -E 'virtio|vmw|xen|hv_' | sort || true", "lspci -k 2>/dev/null | grep -A3 -Ei 'virtio|vmware|xen|hyper-v' | head -20 || true"},
			{"cloud-init status 2>/dev/null || true; find /etc/cloud -maxdepth 2 -type f -printf '%p\\n' 2>/dev/null | sort | head -16", "{ cloud-init query --all 2>/dev/null || true; } | head -24; systemd-detect-virt 2>/dev/null || true"},
		},
	}
	items, ok := probes[objectiveID]
	if !ok || order < 1 || order > len(items) {
		return "", "", false
	}
	item := items[order-1]
	return item.diagnostic, item.transfer, true
}
