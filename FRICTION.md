# Friction Log

## Nits
1. a custom RBAC policy gives a raw JSON editor with no indication of what should go in there.   also the perms all prefix with KOTS which is legacy
2. it was not obvious in the docs what permissions are needed for actions.  I finally found what I needed in the swagger reference.
3. Team settings should be on the left where other UX concerns belong.  It doesn't fit between Support and Docs and I honestly forgot it was there and couldn't find RBAC settings for a minute
4.  uploading a support bundle:  it says instance not found, but there's an instance id shown.  The helper tabs that are supposed to display information show nothing, but opening the files shows tons of information. 


## Larger Issues

### SDK subchart with Helm alias doesn't receive injected license

  - Trying to do: Install GameShelf via Helm CLI with the Replicated
   SDK as a subchart, aliased as gameshelf-sdk for branding (rubric
  requirement 2.1 says deployment must be named <your-app>-sdk)
  - Expected: The Replicated registry injects license data into the
  chart values at pull time. The SDK subchart should receive this
  data and start successfully.
  - Actual: The SDK crashed with "either license in the config file
  or integration license id must be specified." The registry injects
   values under replicated: but the subchart is aliased as
  gameshelf-sdk, so Helm routes those values to the wrong key. The
  SDK never sees its license.
  - Resolution: Had to manually pass --set
  gameshelf-sdk.integration.enabled=true --set
  gameshelf-sdk.integration.licenseID=<id> to work around it. Took
  ~45 minutes of debugging to identify the alias as the root cause.
  The helm pull --untar command was key to seeing that the license
  was injected but not reaching the subchart.
  - Severity: Blocker — the rubric requires branding the SDK
  deployment name, which requires an alias, which breaks automatic
  license injection. There's no documentation covering this
  interaction.

### SDK integration mode loses release context

  - Trying to do: Get the SDK working after the
  alias broke automatic license injection
  - Expected: The integration mode workaround
  would provide full SDK functionality
  - Actual: Integration mode only provides
  license data. The SDK loses release context —
  app version shows --- in vendor portal, update
   checks can't work (SDK doesn't know current
  version to compare against). Custom metrics
  work, license gating works, but version
  reporting and update availability are broken.
  - Resolution: No resolution — this is a
  downstream consequence of the alias/license
  injection issue. A normal install without the
  alias would work correctly, but the rubric
  requires branding the SDK deployment name.
  - Severity: Blocker for rubric items 2.6
  (update banner) and 2.9 (instance version
  reporting). These features cannot work
  properly in integration mode.


## summary of chat history M-W 
1. CMX Tunnel + Cloudflare CNAME conflict — Blocker. Custom domain
   can't point to CMX because replicatedcluster.com is behind
  Cloudflare. ~1 hour, no resolution.
  2. Image proxy path format underdocumented — Annoyance. Had to
  discover app slug requirement and index.docker.io vs docker.io
  through trial and error. ~30 min.
  3. Proxy requires explicit registry linking for public images —
  Annoyance. Expected transparent pass-through, but Docker Hub
  needed to be linked with credentials. ~15-20 min.
  4. SDK subchart alias breaks license injection — Blocker. Aliasing
   the SDK (required for branding) prevents registry-injected
  license from reaching it. ~45 min.
  5. Helm install instructions don't mention imagePullSecrets —
  Annoyance. Registry injects dockerconfigjson into values but
  nothing creates the Secret or documents this step. ~20-30 min.
  6. Bitnami chart version pinning → non-existent image tags —
  Annoyance. Not strictly Replicated, but a common pitfall. ~15 min.
  7. ARM vs AMD64 platform mismatch on CMX — Annoyance. No
  documentation about expected platform. ~10 min.
  8. No end-to-end "first deploy" walkthrough — "I would have
  churned." Each issue above was discovered independently through
  failure. ~1.5 hours cumulative.


## Items I skipped and (may or not have) returned to
 1. Tier 0 — TLS/cert-manager — template exists but couldn't test due to  
  CMX tunnel + Cloudflare CNAME conflict. Deferred to embedded cluster
  where it should work.                                                    
  2. Tier 2.6 — Update available banner — skipped, claiming SDK integration
   mode (aliased as gameshelf-sdk) breaks version reporting/update checks. 
  May be worth retesting.
  3. Tier 2 — Optional ingress — template exists but untested on CMX.      
  4. Tier 3 — External DB preflight analyzer — runPod collector output path
   doesn't match textAnalyze fileName. Never resolved.                     
  5. Tier 3 — Support bundle health + DB analyzers — just fixed the        
  fileNames, not yet re-verified on CMX.                                   
  6. Duplicate preflight: block in values.yaml — cosmetic, needs cleanup.
  7. Word search game bug — broken in browser.                             
                  

## Airgap friction

### Airgap bundle 400 with correct credentials and customer settings
- Trying to do: Download EC airgap bundle via `curl https://replicated.app/embedded/gameshelf/unstable/0.1.64?airgap=true`
- Expected: Bundle downloads once customer has `airgap=true` and `isEmbeddedClusterDownloadEnabled=true`
- Actual: HTTP 400, even with both customer flags enabled and valid license auth
- Resolution: Two separate settings required that are not clearly linked in docs:
  1. **Channel-level**: Vendor Portal → Channels → Edit → "Automatically create airgap builds for newly promoted releases" must be enabled
  2. **Customer-level**: "Airgap Download Enabled" on the license
  Without the channel setting, no airgap bundle is ever built regardless of customer settings.
  Alternatively: go to Releases → find the release → manually trigger airgap bundle build.
- Severity: Blocker. Customer setting alone is insufficient and misleadingly named. Spent ~30 min debugging with CLI inspection before finding the channel setting.
- Also: URL format without version (`/embedded/gameshelf/unstable?airgap=true`) works; with explicit version label also works once bundle is built.

## EC3 friction
1.  if you don't have configurations set up, the whole thing is stuck and the text reporting the error is very faint and hard to read.
2.  it's not clear why I need a kots yaml file for an EC install
3.  preflights on a fresh VM fail fast.  It seems there's easily scripted fixes we could provide to streamline the onboarding
4.  Host preflights run during in-place upgrades and fail on ports already in use by the running cluster (e.g. etcd on 2379/TCP). These checks should be skipped for upgrades since the cluster is already installed — flagging in-use ports as blockers on an existing node is incorrect behavior and blocks the upgrade flow in the web UI.
5.  In-place upgrade via the web UI fails because EC3 attempts to reinstall k0s rather than upgrade the existing cluster. The UI shows "installation" state during what should be an upgrade flow, then errors on k0s already being present. Upgrade appears to not be supported or is broken in this EC3 alpha version.
6.  **In-place upgrade via web UI fails; headless upgrade works** — The EC3 web UI upgrade flow fails (see items 4 and 5). Running `sudo ./gameshelf upgrade` headlessly succeeds. Additionally, the headless upgrade on beta.1 fails with `daemon binary not found: expected /usr/local/bin/assets/gameshelf-service` — the binary exists at `/usr/local/bin/gameshelf-service` but beta.1 expects it in an `assets/` subdirectory. Workaround: `sudo mkdir -p /usr/local/bin/assets && sudo ln -s /usr/local/bin/gameshelf-service /usr/local/bin/assets/gameshelf-service`. Severity: Blocker for web UI upgrade path; headless path requires a manual symlink workaround.
7.  **`ReplicatedImageRegistry` docs don't distinguish behavior by image origin** — The docs show using `ReplicatedImageRegistry (HelmValue ".replicated.image.registry")` to handle the replicated SDK image for airgap. This implies the function works uniformly for all images regardless of origin. In practice, calling `ReplicatedImageRegistry` on `proxy.replicated.com` (the SDK's native registry) in beta.1 does not work: without `noProxy=true` it produces a doubled proxy URL (`proxy.replicated.com/proxy/gameshelf/proxy.replicated.com/...`); with `noProxy=true` it returns an empty string for online installs, causing the image to resolve to `docker.io/library/replicated-sdk-image` (wrong). The SDK chart already uses `proxy.replicated.com` natively and works without any `helmchart.yaml` override — EC3 handles routing for it automatically. The docs don't make clear that the replicated SDK image is a special case that should be left alone. Resolution: removed the `replicated.image.registry` override entirely. ~3 hours of debugging across multiple PRs.


