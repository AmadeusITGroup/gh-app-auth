# Architecture Decision Records (ADR)

This directory records the significant technical decisions made in `gh-app-auth` — what was decided,
why, and what it costs. An ADR is a short, immutable document written at the moment of the decision.
It is not a design doc to keep updated: when a decision changes, write a new ADR that supersedes the
old one.

## When to write an ADR

Write one when a change is hard to reverse or when a future reader would reasonably ask "why is it
done this way?".

**Write an ADR for:**

- Breaking changes — CLI flag or command removals/renames, config file format changes, credential
  helper protocol changes
- New features that introduce a concept, storage location, or trust boundary (a new credential type,
  a new secret backend, a new authentication flow)
- Security-relevant decisions — token lifetimes, key storage location, permission checks, what is
  logged and how it is redacted
- Technology choices — adding or replacing a dependency, changing the keyring library, changing the
  packaging tool
- Significant refactoring that changes the shape of a package or the boundary between packages
- Deliberately rejecting an obvious alternative (record why, so it is not relitigated)

**Skip the ADR for:**

- Bug fixes that restore documented behaviour
- Adding tests, docs, or comments
- Formatting, linting, and dependency version bumps with no behavioural change
- Internal renames with no API or behaviour impact

If you are unsure, err toward writing one. An unnecessary ADR costs ten minutes; a missing one costs
an archaeology session.

## Naming and numbering

```text
docs/adr/NNNN-short-title.md
```

- `NNNN` is a zero-padded, monotonically increasing number. Take the next unused one; never reuse a
  number, even for an abandoned ADR.
- `short-title` is lowercase and hyphen-separated, matching the title.

Example: `docs/adr/0002-store-pats-in-os-keyring.md`

## Required sections

| Section | Content |
|---------|---------|
| **Title** | `# NNNN Short Descriptive Name` — state the decision, not the problem |
| **Date** | `Date: YYYY-MM-DD` — when the decision was made |
| **Status** | `Proposed`, `Accepted`, `Deprecated`, or `Superseded by NNNN` |
| **Context** | Why the decision was needed: the forces, constraints, and alternatives considered |
| **Decision** | What was decided, in the active voice ("We will…"), plus the concrete change |
| **Consequences** | The trade-offs — both the benefits and the costs, including what still does not work |

Keep it factual. Reference the packages, files, and constants involved so the ADR stays anchored to
the code. Include the alternatives you rejected and why: that is usually the most valuable part.

## Status lifecycle

```text
Proposed ──► Accepted ──► Deprecated
                 │
                 └──────► Superseded by NNNN
```

- **Proposed** — under discussion, typically in an open PR
- **Accepted** — the decision is in effect and reflected in the code
- **Deprecated** — no longer applies, with no direct replacement
- **Superseded by NNNN** — replaced by a later ADR; keep the original text intact and add the pointer

Never rewrite the body of an `Accepted` ADR to reflect a new decision. Change its status and write
the new one.

## Template

Copy this into a new file and fill it in.

````markdown
# NNNN Title Of The Decision

Date: YYYY-MM-DD
Status: Proposed

## Context

What problem forced a decision? Include the relevant constraints: GitHub API behaviour, security
requirements, platform differences, backward-compatibility obligations. Point at the code involved
(`pkg/...`, `cmd/...`).

Alternatives considered:

- **Alternative A.** Why it was rejected or set aside.
- **Alternative B.** Why it was rejected or set aside.

## Decision

What was decided, stated plainly. Show the concrete change — the new flag, the constant, the
interface, the file layout:

```go
// Illustrative snippet of the decided approach
```

## Consequences

- What improves as a result.
- What it costs: added complexity, new failure modes, performance impact.
- What deliberately still does not work, and why that is acceptable.
- Migration or backward-compatibility impact for existing users and configs.
- How the decision is protected against regression (which test enforces it).
````

## Index

| ADR | Title | Date | Status |
|-----|-------|------|--------|
| [0001](0001-jwt-clock-skew-margin.md) | Leave Margin on GitHub App JWT Timestamp Claims | 2026-08-05 | Accepted |

Add a row here whenever you add an ADR.

## Further reading

- [AGENTS.md](../../AGENTS.md#architecture-decision-records-adr) — when AI agents must propose an ADR
- [CONTRIBUTING.md](../../CONTRIBUTING.md) — contribution workflow and commit conventions
- Michael Nygard's [original ADR article](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions.html)
