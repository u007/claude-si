# Perl golden fixtures

These fixtures characterize `okf-update.pl` from commit `f74ffe0` before the Go replacement.

Coverage:

- Root and nested conforming documents.
- Missing, empty, quoted, and multiline titles and descriptions.
- Literal and folded scalars.
- Deprecated documents.
- Non-conforming documents.
- Hidden files and directories.
- Default exclusions and `.okfignore` file and directory patterns.
- LF and CRLF input.
- Tied titles, Unicode titles, and deterministic path ordering.
- Existing and missing logs.
- Existing and missing date headings.
- Creation, update, deprecation, and deletion actions.

`{{LOCAL_DATE}}` marks the local capture date in input and expected log files. Tests substitute a fixed date before comparison. `.okf.lock` is excluded because it is an empty synchronization artifact.

Regenerate only on a host with Perl, using the script from commit `f74ffe0`. Windows CI reads these committed expectations and never invokes Perl.
