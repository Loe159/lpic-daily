#!/usr/bin/env bash
set -euo pipefail
# This scenario can ONLY be solved through the actual GRUB serial console.
# A shell script executed inside a running guest cannot retroactively modify
# the bootloader commands or the command line of the already-running kernel.
cat <<'INSTRUCTIONS'
1. Dans le terminal LPIC Daily, lancer :boot-menu, puis :console immédiatement.
2. À l'écran GRUB 2, presser c pour accéder à l'invite grub>.
3. Taper : set lpic_grub_cli=verified
4. Taper : save_env lpic_grub_cli
5. Presser Échap pour revenir au menu, puis e sur l'entrée Fedora normale.
6. À la fin de la ligne qui commence par linux, ajouter lpic.console_probe=1.
7. Démarrer avec Ctrl-X ; une fois Fedora revenu, sortir avec Ctrl-].
8. Saisir :check dans le terminal LPIC Daily.
INSTRUCTIONS
# Return nonzero deliberately. This is a MANUAL console reference procedure,
# not a misleading automated solution that claims the checks passed.
exit 64
