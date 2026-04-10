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
                  
  Items 1-4 are rubric-relevant. Want to circle back on any of these before
   moving to Tier 4, or press forward?



## EC3 experience
1.  AI found docs saying we could provision an embedded cluster instance and install, but that's not supported for v3.  you just need a VM somewhere and then follow the instructions in the UI.  the docs probably need those instructions explicitly for CLI users. 
2.  it's confusing that we need KOTS config/spec in EC.
