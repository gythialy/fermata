# Fermata documentation

Fermata (停琴) is a menu bar break reminder for macOS. This directory is the
written record of the product: what it does, why it is shaped this way, how it
is built, and how it looks.

The documentation is in English. The user-facing README is bilingual:
[English](../README.md) · [简体中文](../README.zh-Hans.md).

## Start here

| If you want to… | Read |
|---|---|
| Use the app | [README](../README.md) |
| Know what the app must do, exactly | [functional-spec.md](functional-spec.md) |
| Understand the code | [architecture.md](architecture.md) |
| Change a colour, a size or a surface | [design-system.md](design-system.md) |
| Know why a decision was made, or what is next | [product.md](product.md) |
| Build, test, localise or package it | [development.md](development.md) |
| See the visual reference | [product.md §11](product.md#11-visual-canvas) |

## The documents

| Document | Status | Owns |
|---|---|---|
| [product.md](product.md) | Reference | Goals, scope, decisions, milestones, compatibility, budgets, risks, the visual canvas |
| [functional-spec.md](functional-spec.md) | **Normative** | Scheduling, state machine, break and pause rules, settings fields, notifications, CLI, i18n, acceptance cases |
| [design-system.md](design-system.md) | **Normative** | Direction, the mark, design tokens, components, surfaces, cross-platform stance, accessibility, copy |
| [architecture.md](architecture.md) | As built | Package structure, data flow, state ownership, concurrency, tests, build and release |
| [development.md](development.md) | Guide | Prerequisites, make targets, i18n, fonts, icons, sounds, screenshots, animation, GitHub assets, CI and releases |

## Precedence

When two documents disagree, the narrower one wins:

1. **Behaviour** — [functional-spec.md](functional-spec.md). If the app and the
   spec disagree, that is a bug in one of them.
2. **Visuals, sizes, controls and platform shells** —
   [design-system.md](design-system.md).
3. **Code structure** — [architecture.md](architecture.md).
4. **Scope and product decisions** — [product.md](product.md).

The rule is: change the document first, then the code, then the canvas.

## Assets

| Path | What it is |
|---|---|
| [`images/`](images/) | Real screenshots and the break window's loop, rendered from the app's own views by `make screenshots` and `make animation`; English, with the Chinese set in [`images/zh-Hans/`](images/zh-Hans/) |
| [`../.github/assets/`](../.github/assets/) | Banner, social preview, mark and icon for the repository |
| [`../artifacts/fermata-ui-ux-design/`](../artifacts/) | The Open Artifact canvas indexed in [product.md §11](product.md#11-visual-canvas) |

Screenshots and artwork are generated, not traced; see
[development.md](development.md) for the commands.
