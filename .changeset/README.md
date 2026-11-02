# Changesets

Each file here describes one change for the next release. Add one with `bun x changeset`. The app
has a single version (every package is in one `fixed` group), so pick whichever package you changed.
For Go changes, pick `@ledger-01/backend`.

[CONTRIBUTING.md](../CONTRIBUTING.md#versioning-and-changesets) explains what counts as a major,
minor or patch change. The Release workflow turns these files into a "Version packages" PR, and
merging that PR publishes the release.
