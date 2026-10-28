# Grok.com web Imagine routes — REA findings (ready to apply to a grok2api fork)

Source: public grok.com Next.js bundles captured 2026-10-27, local copy at
`~/workspace/grok-capture/` (128 chunks, ~11 MB). Static analysis only unless
marked LIVE. No live grok.com call is claimed here beyond the two tests noted.

## Core finding

Edit and video are NOT separate REST endpoints in the grok.com web client.
Both go through one conversation endpoint:

- `POST /rest/app-chat/conversations/new`
  - body includes `kind: ConversationKindImagine`, `enableImageStreaming: true`,
    `message`, `imageAttachments` / `imageEditUris`, `imagineCanvasContext`,
    `mediaGenInput`
  - follow-ups use `chatAddResponseRaw` on the same conversation
  - image results stream back with `modelResponse.generatedImageUrls`

REA batch 1 (2026-10-28, 12 core Imagine chunks, exit 0, 94 MB evidence,
skipping timeout chunk `01rkw834kdzvs.js`): `ConversationKindImagine` x25,
`mediaGenInput` x14, `/rest/app-chat/conversations/new` x2,
`/ws/imagine/listen` x2. Note: that batch's evidence does not surface the
literal strings `fetchGenerateImageEdits` / `generateVideoForImage` (0 hits),
even though the source chunks contain them (13 / 10 hits in
`1aw97vh8d1x5z.js`). The earlier 2-file key-file REA run did surface them
(x5 / x6). Treat the function names as source-verified, not batch-1-verified.

## Edit

- `fetchGenerateImageEdits` -> `_fetchGenerateImageEdits`
- types: `image_edit` (1 reference) / `multi_image_edit` (multiple)
- inputs: prompt, imageUrls, container/parent post IDs, optional
  `selectionRegions`, `reframe` (expand/zoom + cropRect), `extractSubject`,
  `draw`, aspect ratio / resolution

## Video

- `generateVideoForImage` -> `_runGenerateVideoForImage`
- kinds: `textToVideo`, `imageToVideo`, `referenceToVideo` (first-frame +
  keyframes + last-frame assets), `videoExtension` (with
  `videoExtensionStartTime`)
- inputs: prompt, input assets, mode, `duration`, `imageReferences`,
  `framePins`, `audioAssetIds`
- live updates over websocket `/ws/imagine/listen` on the grok.com host
  (bundle hardcodes `wss://grok-api.gcp.mouseion.dev/ws/imagine/listen`
  only when host contains `mouseion.dev`)

## Ancillary REST (separate, from the same bundles)

- `POST /rest/conversations/files/imagine-asset` — upload reference asset
- `POST /rest/media/video/caption`, `/rest/media/video/music`,
  `/rest/media/video/upscale`
- `POST /rest/media/video-sequence/create`
- `GET /rest/media/imagine/quota_info`, `/rest/media/imagine/generation_estimate`
- `GET /rest/media/post/imagine-feed`
- `GET /rest/models/imagine/overrides`

## New in the 2026-10-28 grep pass (not in the first trace)

- Media pipeline: `/rest/media/pipeline/run`, `/rest/media/pipeline/cancel`,
  `/rest/media/pipeline/resume`, plus template create/fork/publish/save/
  unsave/rollback/list/get/update. This is the Photo Edit template path.
- Media canvas, collection, folder, reference, search, segment:
  `/rest/media/canvas/*`, `/rest/media/collection/*`, `/rest/media/folder/*`,
  `/rest/media/reference/*`, `/rest/media/search/*`, `/rest/media/segment`
- `/rest/media/generate-music`, `/rest/media/generate-voice`,
  `/rest/media/gifs/search|trending`
- `/rest/app-chat/image-generations`, `/rest/app-chat/template-chat`,
  `/rest/app-chat/template-prompt`
- Full media/app-chat path list: `~/workspace/grok-media-rest.txt` (363 lines,
  file-prefixed, dedup by file+path)

## LIVE tests (2026-10-27, browser, logged in as Ruby Hart, drafts only)

- Video: BLOCKED by paywall. 480p, 6s, 2:3, audio off, prompt about a
  golden-hour porch. Dialog "Unlock your creativity with Imagine"
  (`upsell.imagine.title-v4`, generic, not video-specific), plans Plus
  $8/month and Premium $15/month shown. No video, no quota wording.
  No subscription was made.
- Edit: FAILED, server error. Uploaded a simple drawn tree/sky test image,
  Photo Edit template, Background Changer, prompt "change the background
  to sunset". Draft post created, then toast "Server failed to respond"
  after retry, ~2 minutes, no edited image, no quota wording.
- Nothing published, shared, or subscribed in either test.

## Distinct from grok2api's own route

- grok2api: `POST /v1/images/edits`, JSON only, model
  `grok-imagine-image-edit`, data-URI references. Schema-correct on
  2026-10-27, but returned `429 upstream_quota_exhausted` on direct probe.
- Do not conflate the two: the web protocol above is not the grok2api API.

## Code-coverage audit against this fork's Web provider (2026-10-28)

Checked against the fork's `backend/internal/infra/provider/web/` at head
`e2fd7ab` (before this doc's updates). The core REA protocol is ALREADY
implemented upstream — these are not new changes to invent:

- `POST /rest/app-chat/conversations/new` for image edit:
  `image.go` `generateImageEdit` (~line 878) with `buildImageEditPayload`
  (~line 922): `modelName: imagine-image-edit`, `enableImageStreaming`,
  `mediaGenInput.imageToImage` with prompt, inputAssets, aspectRatio.
- Same endpoint for video: `video.go` `videoCreatePayload` (~line 546):
  `modelName: imagine-video-gen`, `kind: CONVERSATION_KIND_IMAGINE`,
  `mediaGenInput.textToVideo` with prompt, aspectRatio, duration,
  resolutionName; `video.go` ~line 278 posts to `conversations/new`.
- Websocket: `image.go` `imagineURL` (~line 1776) sets path
  `/ws/imagine/listen`.
- Quota: `quota.go` ~line 104 uses `/rest/media/imagine/quota_info`.
- Catalog: `catalog.go` lines 27-31 list `grok-imagine-image-lite`,
  `grok-imagine-image`, `grok-imagine-image-2.0`, `grok-imagine-image-edit`,
  `grok-imagine-video`.

NOT implemented in the Web provider (searched, non-test Go files only),
and NOT implemented by this change, because they sit outside grok2api's
OpenAI-compatible `/v1/*` surface and none is verified live here:

- Media pipeline template routes (`/rest/media/pipeline/run|cancel|resume`,
  template create/fork/publish/save/unsave/rollback)
- Canvas / collection / folder / reference / search / segment routes
- `/rest/media/video-sequence/create`, `/rest/media/video/caption`,
  `/rest/media/video/music`, `/rest/media/video/upscale`
- `/rest/media/imagine/generation_estimate`, `/rest/media/post/imagine-feed`,
  `/rest/models/imagine/overrides`
- Video kinds `referenceToVideo` and `videoExtension`, and edit options
  `selectionRegions` / `reframe` — no matches in the provider code
  (as of the audit above).

UPDATE — production code change, pushed to this fork (see commit history):
Web `imageToVideo` and `referenceToVideo` are now implemented in
`backend/internal/infra/provider/web/video.go` (`GenerateVideo`,
`uploadWebVideoAssets`, `videoCreateImagePayload`,
`videoCreateReferencePayload`), with the gateway Web image/reference
block removed in `backend/internal/application/gateway/video.go` and
its test updated. Images are uploaded via the existing V2 direct-upload
path (same as image edit); reference assets serve as both `inputAssets`
and `mediaReferences`, with the first asset as `firstFrameAsset`.
`videoExtension` (needs a grok asset ID, not a URL) and
`referenceAudios` (web expects audio asset IDs, not Console voice IDs)
remain rejected in the Web adapter, with Console as the fallback.
Tests: `backend/internal/infra/provider/web/video_kinds_test.go`
(httptest, data-URI images) plus the full web and gateway packages pass
(`go test ./internal/infra/provider/web/ ./internal/application/gateway/`).
Live upstream video is still NOT proven: this code has not run against
real grok.com (browser UI video is paywalled on the test account), and
the image-upload `file_source` mapping is the one unverified assumption.

UPDATE 3 — live Web video fix, 2026-10-28 (file_source proven, image
attachment fix pending live proof):
The first live run through this code (Ruby Hart, image-to-video, 6s,
720p) failed upstream with `Invalid file_source value:
image_edit_upload` — the code had passed the stage name as the V2
`file_source` field. The correct value is
`IMAGINE_SELF_UPLOAD_FILE_SOURCE`, the same value the image-edit flow
passes; that fix is in `video.go` (`uploadWebVideoAssets`,
`resolveWebVideoExtensionSource`). After that fix the job completed
live (`video_yEw7XJgaer2XOhyYVRiUkEgk`, Web provider, 6s, 720p) but the
output was a generic 1280x720 mountain-lake video with no trace of the
input image and the wrong aspect — upstream accepted the call but
discarded the mediaGenInput asset. The follow-up fix in the same
`video.go` therefore sends the uploaded asset as a top-level
`imageAttachments` entry on the `conversations/new` call (the web
client's inputs include imageAttachments alongside mediaGenInput) and
sets `useFirstFrame: true` explicitly on `imageToVideo`, instead of
leaving it to an omitted-field default. That attachment fix is
NOT yet live-proven: the reference-path retry after the first video
was blocked by `429 upstream_quota_exhausted`, so the next live run
is the proof, not this code change.
Web `videoExtension` is now implemented in the same `video.go`
(`resolveWebVideoExtensionSource`, `videoCreateExtensionPayload`):
a bare asset ID is used as-is, while a URL/data-URI source video is
downloaded (up to 256 MiB) and re-uploaded via V2 direct upload, and
gateway `routesForVideoOperation` now admits Web grok-imagine-video
routes for extend (edit stays Console-only — the web client has no
video-edit kind). Reference-audio jobs are now routed away from Web in
`CreateVideo` (`filterOutWebVideoRoutes`) instead of only failing in
the adapter. The video-upload-via-V2 step is the new unverified
assumption, same class as the image one.

What is still genuinely NOT implemented, and why it is not in this push:
- Image-edit options `selectionRegions` / `reframe` / `extractSubject` /
  `draw` / `multiRegionEdits`: `ImageEditRequest` has no fields for
  them, so adding them would invent a new public API surface, not just
  fill in a provider payload. The bundle field shapes are recorded in
  GROK-WEB-ROUTES.md's route inventory for when that API is designed.
- Post-generation transforms (`/rest/media/video/caption|music|upscale`,
  `/rest/media/video-sequence/create`, `generation_estimate`,
  `imagine-feed`, `models/imagine/overrides`, pipeline/template/canvas/
  collection/folder/reference/search/segment): no matching /v1 endpoint
  exists in grok2api, so these need new endpoints, not a provider patch.
- Live upstream proof for any Web video kind: still requires a live
  call approval, and browser UI video remains paywalled on the test
  account.

Companion file in this fork: `grok-media-rest-paths.txt` — 178 unique
media/app-chat `/rest/` paths (deduped from the 363 file-prefixed lines
in the local extraction), for reference only.

## What applying this to a fork means (and does not mean)

- This document is the portable change set. It does not, by itself, implement
  a grok.com-conversation provider in grok2api's Go backend, and no such
  implementation is verified end-to-end here, because live video is paywalled
  and live edit failed server-side on this account.
- Safe local file to also add to a fork: `start-grok2api.sh` only.
- NEVER add: `ADMIN-CREDENTIALS.txt`, the compiled `grok2api` binary,
  `config.yaml`, `data/`, or logs — all excluded by .gitignore or by policy.
