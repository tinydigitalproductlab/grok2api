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
  `selectionRegions` / `reframe` — no matches in the provider code.
- Live video is paywalled and live edit failed server-side on the test
  account, so implementing any of the above from static bundle evidence
  alone would be unverified. They are recorded here as future work, not
  as working code.

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
