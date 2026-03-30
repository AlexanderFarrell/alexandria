Alexandria 📚



Goal

A self-hosted reading and media server that makes your personal library a first-class reading experience — with full epub rendering, text-to-speech, and deep integration with your learning goals.



Problem

Calibre-web is adequate for library management but blocks the actual reading experience: no full epub rendering in-browser, no TTS for listening while commuting or working, no integration with your goal of reading 50 books in 2026. Your reading infrastructure is load-bearing for your #1 annual initiative and it's currently broken.



Users

Primarily you. Potentially family. Built for deep personal use, not public deployment.



Principles

Reading is first-class. TTS and full epub rendering are core features, not plugins or afterthoughts.

Local-first. All content lives on your infrastructure. No cloud dependency for access.

Fast to open a book. From login to reading in under 10 seconds.

Calibre-compatible. Imports existing Calibre library without manual re-entry.

Integrated with Zealot. Books can be linked to Education tickets, reading progress trackable.



Current State

Calibre-web running, library populated

No epub rendering, no TTS

AX01 (scaffold) and AX02 (specify) are the only child tickets — project is pre-foundation



Milestone Map

v0.1 — Scaffold & Spec: Project structure, tech decisions, data model defined

v0.5 — Core Reader: Epub rendering in-browser, library browsing, basic auth

v0.8 — TTS & Audio: Text-to-speech pipeline, playback controls, resume position

v1.0 — Zealot Integration: Books linkable to items, reading progress as tracker entries, 50-book goal visible



Active Epic

AX01 — Scaffold Project



Dependencies

Citadel (hosting)

Zealot (integration target)