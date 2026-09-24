# Curriculum language and terminology style

## Language
Write learner-facing explanations in French.

Do not force awkward translations of established Linux vocabulary. Preserve commands, flags, paths, protocol names, systemd object types and conventional English terms when that is the clearest form.

Good examples:
- `pipe`
- `filesystem`
- `systemd unit`
- `target`
- `socket`
- `stdout` / `stderr`
- `mount point`
- `routing table`

When useful, explain the meaning in French on first use rather than replacing the technical term.

## Syllabus labels
Content that discusses tools/practices should identify its role when relevant:
- **LPIC requis** (`lpic-required`)
- **LPIC legacy** (`lpic-legacy`)
- **Pratique moderne** (`modern-practice`)

An item may carry more than one label if, for example, a modern command is also explicitly required by LPIC.

## Progressive teaching
First exposure should answer:
- what problem does this concept solve?
- what is the smallest correct mental model?
- what one useful operation can the learner perform now?

Later material adds options, internals, failure cases, distribution differences and cross-topic incidents.

## Avoid
- unexplained command dumps;
- exhaustive man-page reproduction;
- unnatural French translations of command/tool terminology;
- teaching one distro's defaults as universal Linux behavior;
- presenting a legacy command as the modern recommendation without context;
- wording copied/adapted from restricted LPI learning materials.
