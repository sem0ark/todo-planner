## Vision
**Personal sustainability planner** for designing, executing, and refining versioned "ideal-day" templates across life states and seasons.

The app operates across two distinct surfaces that serve different moments of the day:
- **Live surface** — a persistent, minimal presence always available during the day. Captures activity transitions and confirmations with minimal interruption, without requiring the user to navigate or context-switch.
- **Review surface** — an intentionally opened environment for reconstructing, adjusting, and analyzing past days. Supports both light correction of live-logged data and full retroactive reconstruction when live logging did not occur. Opened intentionally, not during active work.

The mental model: Microsoft Clock (live, minimal, always present) + Obsidian Day Planner (timeline editing and review) - two separate surfaces, not one combined view.

The core question the app answers over time: "Is the life structure I designed for myself actually sustainable?"

## Goals
- **Design and version sustainable day structures** per life state and season. Templates are independent, named, and maintained in a personal library. "Create From" workflow allows iterative refinement without starting from scratch.
- **Capture reality with near-zero friction** - the live surface is the primary input mechanism, designed to stay out of the way during active work and surface only at natural transition points. The review surface provides a safety net: any day can be reconstructed/adjusted after the fact.
- **Make untracked time and unlogged days a visible signal** - gaps in the record are never silently ignored or treated as neutral. They are surfaced as potentially meaningful data, distinct from rest and distinct from intentional blank periods the user cannot recall.
- **Reveal template health over time** - the app accumulates the gap between planned structure and lived reality, and makes that gap legible: per-template, per-category, and across time windows the user selects. The goal is to surface whether a template is realistic, not to judge whether the user followed it.
- **Account for personal seasonal and contextual variation** - life state and season are first-class dimensions of the planning system. Templates reflect individual patterns (energy, schedule, environment) rather than general norms, and the user maintains distinct versions for distinct contexts.

## TODO
- [x] Think on what to do with the application, speficially the requirements and main goals I want to achieve.
- [x] Plan out the main interaction flow of the widget.
- [x] Plan out the API structure -> will be just a glue to DB, so that we could use it as just a sync engine, better to still not expose it to much though...
- [x] Plan out the web UI and dashboard flow.
- [x] Plan out the phone UI widget-like flow + how to later migrate web UI.
- [ ] UI analytics logic.
- [ ] More reselient day event logging logic.
