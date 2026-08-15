# Security Policy

## Reporting a Vulnerability

This library is read-only: it opens SQLite databases in `mode=ro` and never
writes. The attack surface is small (file reads, JSON parsing, one exec
call for the CLI fallback).

To report a vulnerability, open a private GitHub Security Advisory
(GitHub > Security > Advisories > New draft advisory). Include a
minimal reproduction and the affected version. You will receive a
response within 72 hours.

## Scope

- **In scope**: any input that causes a panic, crash, or unexpected file
  read in the library's public API (`Open`, `OpenContext`,
  `DiscoverProjects`, `ParseProjectsOutput`, `DecodeParts`).
- **Out of scope**: the Crush application itself (report upstream), the
  `modernc.org/sqlite` driver (report upstream), behavior documented as
  a known limitation in `extractJSONObject`'s comment.
