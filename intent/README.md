# Intent home

This directory is the Plan-stage home for the [AI-Native SDLC](https://claude.com/blog/the-ai-native-sdlc-playbook) loop encoded in [`AGENTS.md`](../AGENTS.md).

Each change lives in `intent/<slug>/` and accumulates committed artifacts the next stage reads:

1. `intent.md` — Plan (what, why, constraints)
2. `spec.md` — Design (requirements and design from the accepted intent)
3. `plan.md` — Build (files, order, tests, risks); accepted before code

Copy the files in [`_templates/`](_templates/) into a new `intent/<slug>/` directory. Do not edit the templates in place for a real change.

The founding product record, reverse-documented from the shipping tree, is [`harnezpad/`](harnezpad/).

Later stages commit the diff and its tests (Build/Test), the pull request and review findings (Deploy), and — when production breaks an expectation — a new `intent.md` (Maintain).
