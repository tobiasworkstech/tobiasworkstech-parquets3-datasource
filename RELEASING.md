# Releasing

End-to-end checklist for shipping a new version of `tobiasworkstech-parquets3-datasource`.

The plugin is approved and live in the [Grafana plugin catalog](https://grafana.com/grafana/plugins/tobiasworkstech-parquets3-datasource/). Most of the work is automated — the only **mandatory** manual step is submitting the update on Grafana's plugin portal.

## Overview

```
bump version  ─►  push tag  ─►  CI signs zip  ─►  publish GitHub release  ─►  upload to Grafana portal
   (local)        (local)       (auto)             (one gh command)         (manual, one form)
```

## 1. Pre-flight (local)

```bash
# clean working tree, all reviewed changes committed
git status
git pull --ff-only

# sanity checks
npm run lint
npm run test:ci
npm run build
go test ./...
mage -v   # builds all five 64-bit platforms
```

Verify a CHANGELOG entry for the new version exists in `CHANGELOG.md` (one section per release, with **Bug Fixes** / **Improvements** / etc. as needed).

## 2. Bump version and push tag

```bash
# choose patch / minor / major
npm version patch    # 1.2.16 -> 1.2.17

# pushes the version-bump commit and the new v* tag
git push origin main --follow-tags
```

`npm version` already creates both the commit and the annotated tag. The `--follow-tags` flag pushes both in one go.

## 3. Watch CI (auto)

The `v*` tag triggers `.github/workflows/release.yml`:

```bash
gh run watch --repo tobiasworkstech/tobiasworkstech-parquets3-datasource \
  $(gh run list --workflow Release --limit 1 --json databaseId --jq '.[0].databaseId')
```

The `grafana/plugin-actions/build-plugin@main` action does:

1. Build frontend (`npm run build`)
2. Build backend for all platforms (`mage buildAll`)
3. Run the OSV-Scanner CVE check + plugin validator
4. Call Grafana's signing API → embeds `MANIFEST.txt`
5. Generate SLSA build-provenance attestation
6. Attach the signed zip + `.sha1` to a **draft** GitHub release

If it fails, fix forward (new commit + `npm version patch` again). Common breakages:

| Failure | Cause | Fix |
|---|---|---|
| `Field is required: rootUrls` (HTTP 409) at signing step | `GRAFANA_ACCESS_POLICY_TOKEN` is a private-plugin token | Regenerate as a community-plugin token, update the GitHub secret |
| OSV-Scanner reports HIGH CVE | Vulnerable transitive dep | `go get <pkg>@<fixed>` or add an `overrides` entry in `package.json` |
| `math.MaxUint32 overflows int` on linux/arm | thrift v0.23 doesn't build on 32-bit | Drop the platform in `Magefile.go` (already done) |
| Validator: `no-ident-root-dir` | Zip root is `dist/` instead of the plugin id | The action handles this; only matters for manual zipping |

## 4. Publish the GitHub release

`build-plugin` creates the release as a **draft**. Flip it to public + latest:

```bash
VER=$(node -p "require('./package.json').version")
gh release edit "v${VER}" \
  --repo tobiasworkstech/tobiasworkstech-parquets3-datasource \
  --draft=false --latest
```

This makes the zip URL stable so Grafana's validator can fetch it.

## 5. Submit to Grafana (manual, ~1 minute)

Grab the MD5 of the signed zip:

```bash
VER=$(node -p "require('./package.json').version")
ZIP="tobiasworkstech-parquets3-datasource-${VER}.zip"
gh release download "v${VER}" \
  --repo tobiasworkstech/tobiasworkstech-parquets3-datasource \
  --pattern "${ZIP}" --dir /tmp --clobber
md5 -q "/tmp/${ZIP}"
```

Open <https://grafana.com/auth/sign-in> → **My Plugins** → find `tobiasworkstech-parquets3-datasource` under **Published Plugins** → **Update Plugin**.

| Field | Value |
|---|---|
| Plugin version | `1.2.x` (matches the tag) |
| Plugin zip URL | `https://github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource/releases/download/v1.2.x/tobiasworkstech-parquets3-datasource-1.2.x.zip` |
| MD5 | output of `md5 -q` above |
| Source URL | `https://github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource` |
| Submission notes | Bullet-list of changes (link the relevant CHANGELOG section if it's long) |

Click **Submit**. Status flips to **Received**.

Since the plugin is already approved and listed, only the automated validator runs on update submissions — no human review. Approval is typically minutes-to-hours and silent: the new version simply appears on the [catalog page](https://grafana.com/grafana/plugins/tobiasworkstech-parquets3-datasource/).

## 6. After the new version is live

When the new version is published in the catalog, users can install/upgrade via:

```bash
grafana-cli plugins update tobiasworkstech-parquets3-datasource
```

or by setting `GF_INSTALL_PLUGINS=tobiasworkstech-parquets3-datasource@<version>` on Docker.

## Manual zip (fallback when CI signing breaks)

If `release.yml` can't sign for some reason and you need to ship anyway:

```bash
npm run build
mage -v

VER=$(node -p "require('./package.json').version")
ID="tobiasworkstech-parquets3-datasource"
ZIP="${ID}-${VER}.zip"

rm -f "${ZIP}"
cp -r dist "${ID}"
zip "${ZIP}" "${ID}" -r
rm -rf "${ID}"
md5 -q "${ZIP}"

# upload to GitHub Release manually
gh release create "v${VER}" "${ZIP}" \
  --repo tobiasworkstech/tobiasworkstech-parquets3-datasource \
  --title "Parquet-S3-Datasource v${VER}" \
  --latest
```

The zip from this fallback is **unsigned** — Grafana will sign it during the catalog submission step (which then becomes mandatory rather than optional).

## Notes

- `.config/` is owned by `@grafana/create-plugin` — never edit it. If the build process drifts from the template, update via `npx @grafana/create-plugin@latest update`.
- Never commit files containing `valdemarpavesi`. Everything in committed code uses `tobiasworkstech`.
- Build-provenance attestations are visible at `https://github.com/tobiasworkstech/tobiasworkstech-parquets3-datasource/attestations`.
