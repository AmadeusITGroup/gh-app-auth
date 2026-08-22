# 0001 Leave Margin on GitHub App JWT Timestamp Claims

Date: 2026-08-05
Status: Accepted

## Context

GitHub App authentication starts with a self-signed JWT that is exchanged for an installation
access token. GitHub validates the JWT's `iat` and `exp` claims against **its own** clock and
rejects the request with HTTP 401 if either falls outside the allowed range:

- `exp` may be at most 600 seconds (10 minutes) ahead of GitHub's current time.
- `iat` may not be in GitHub's future.

`pkg/jwt/generator.go` originally built the claims at the exact boundaries:

```go
now := time.Now()
"iat": now.Unix(),
"exp": now.Add(10 * time.Minute).Unix(),
```

This leaves zero tolerance for clock skew. If the machine generating the JWT runs even one second
ahead of GitHub, `exp` lands at 601 seconds and every API call fails with:

```text
GitHub API returned status 401: {"message":"'Expiration time' claim ('exp') is too far in the future"}
```

The same fast clock simultaneously pushes `iat` into GitHub's future, which triggers a sibling 401
on the `'iat'` claim.

Alternatives considered:

- **Query an NTP server for authoritative UTC.** Rejected. It adds a dependency (Go has no stdlib
  NTP client) and a UDP/123 round trip on the per-operation credential path. Port 123 is commonly
  blocked in exactly the corporate and CI networks where drift occurs, so it would need a fallback
  to the local clock anyway. Unauthenticated NTP is also spoofable, which is unwelcome extra attack
  surface in an authentication tool.
- **Derive the offset from GitHub's `Date` response header.** Considered and set aside. GitHub's
  clock is the authoritative reference and the header costs no extra request, but consuming it
  requires detecting the 401, measuring the offset, realigning JWT generation, and retrying — a
  materially more complex control flow through security-critical code than the problem warrants.
  It remains a viable follow-up if drift beyond the fixed margin proves common.

## Decision

Generate JWT claims with explicit margin on both ends instead of at GitHub's exact limits, using
named constants in `pkg/jwt/generator.go`:

```go
clockSkewTolerance = 60 * time.Second  // backdates iat
jwtValidity        = 9 * time.Minute   // 540s, 60s under GitHub's cap

now := time.Now()
"iat": now.Add(-clockSkewTolerance).Unix(),
"exp": now.Add(jwtValidity).Unix(),
```

The JWT is used only to immediately mint an installation token, so shortening its lifetime from
600 to 540 seconds has no practical cost.

Additionally, `auth.FormatAPIStatusError` (`pkg/auth/apierror.go`) recognises the `exp`/`iat`
rejection messages and appends a hint to synchronise the system clock. It replaces the inline
status-error formatting at all six call sites across `pkg/auth/authenticator.go`, `cmd/setup.go`,
`cmd/debug.go`, and `cmd/test.go`, including the installation ID auto-detection path where the
failure was first reported.

## Consequences

- Clock drift up to 60 seconds in either direction is tolerated, covering the common container and
  CI case that previously failed outright.
- Drift beyond 60 seconds still fails, by design. The corrected margin cannot substitute for a
  synchronised host, so the improved error message now directs users to fix NTP on the machine.
- The JWT is valid for 540 rather than 600 seconds. No caller depends on the longer lifetime; the
  token is exchanged for an installation token immediately after generation.
- The 60-second `iat` backdate means a JWT is nominally valid slightly before it was issued. This
  matches GitHub's own documented recommendation and is bounded by the short overall lifetime.
- The `exp - iat` window is exactly 600 seconds. GitHub evaluates the claims independently against
  its own clock rather than validating the window itself, so this is safe, and it is 60 seconds
  tighter than the window GitHub's documentation suggests.
- Two magic numbers now govern skew tolerance. They are named constants with rationale in comments,
  and `TestJWTToleratesClockSkew` asserts the resulting claims survive simulated drift of 0, 1, 5,
  30, and 60 seconds, so a future change that reintroduces the boundary values fails the build.
