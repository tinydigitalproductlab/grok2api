# Video, figured out — grok.com web client vs grok2api Web provider

## Short answer

Video is NOT a separate video API in the web client. It is the same
conversation call as images, with a different `mediaGenInput` branch:

`POST /rest/app-chat/conversations/new`
- `kind: CONVERSATION_KIND_IMAGINE` (bundle: `ConversationKindImagine`)
- `modelName: imagine-video-gen`
- `message: "<prompt> --mode=<mode>"` (grok2api video.go uses `--mode=custom`)
- `enableImageStreaming: true`, `enableSideBySide: true`,
  `sendFinalMetadata: true`
- `mediaGenInput.<kind>: { prompt, aspectRatio?, duration?, resolutionName?,
  inputAssets?, skipAudio?, ... }`
- Result streams back as `result.response.streamingVideoGenerationResponse`
  with `progress`, `videoPostId` / `videoId`, and finally `videoUrl` /
  `contentUrl` / `assetUrl` (must end `.mp4` or contain `/content` in
  grok2api's parser), plus `fileAttachments` fallback. grok2api also
  accepts SSE (`data:`) or plain JSON stream.

Live updates for image generation use `/ws/imagine/listen`; video in
grok2api is read from the `conversations/new` response stream itself.

## The four kinds (bundle, media-store-v2, 1aw97vh8d1x5z.js)

All four go through `generateVideoForImage` in the web client:

1. `textToVideo` — `{ prompt, aspectRatio?, duration?, resolutionName?,
   skipAudio? }`. No input asset.
2. `imageToVideo` — `{ prompt, inputAssets: [asset], aspectRatio?,
   duration?, skipAudio?, useFirstFrame? }`. `useFirstFrame: false`
   maps internally to mode `r2v`, otherwise `i2v`.
3. `referenceToVideo` — `{ prompt, inputAssets, audioAssets?,
   mediaReferences?, aspectRatio?, duration?, skipAudio?,
   firstFrameAsset?, lastFrameAsset?, keyframes?, gifAsset? }`.
   First-frame + keyframes + last-frame, plus optional GIF source asset.
4. `videoExtension` — `{ prompt, inputAssets: [sourceVideoAssetId],
   duration?, videoExtensionStartTime?, skipAudio? }`. Extends an
   existing video from a start time.

Modes seen in the bundle: `custom` (default for all four in the
`generateVideoForImage` call sites), `fun`, plus UI modes
`spicy` / `loop` / `quick-animate` as Imagine mode labels. The
`--mode=` suffix is appended to the message text, not a separate
top-level field in the grok2api payload.

Audio: `skipAudio` is per-kind (text / image / reference / extension).
The client also has `videoAudioEnabled` / `setVideoAudioEnabled` and
`audioAssets` / `audioInputAssetIds` on reference videos. grok2api's
Web video path does not expose audio at all — it is text-to-video only.

## Durations, resolutions, aspect ratios

Web UI observed live (2026-10-27, Ruby Hart account, before paywall):
Video 480p (selected) / 720p; durations 6s (selected) / 10s / 15s;
aspect 2:3 Tall (selected), Auto, 3:2, 1:1, 9:16, 16:9; audio toggle off.

Code limits:
- grok2api Web: duration must be 1–15s (`videoSegments` returns nil
  outside that); default resolution `720p` if none given.
- Free/Basic Web accounts: duration is clamped to the free cap —
  default 6s, range 1–15s (`settings.go`: `DefaultWebFreeVideoDurationCap
  = 6`, `Min = 1`, `Max = 15`). This is explicitly to avoid upstream
  429s from >6s on free accounts.
- Bundle: duration options come from server `imagine-configs`
  (`videoDurationOptions`), snapped to the nearest allowed value, with
  a `imagine_video_setting_downgraded` event when a setting is cut.
  1080p exists in the bundle (including an `imagine-1080p-upscale`
  path) but is gated — 1080p generation is not the default video path,
  and `allowedVideoResolution` falls back by subscription.
- grok2api README catalog (Web): `grok-imagine-video` = Basic for 720p,
  Super for 480p. That is the opposite of the usual "higher tier =
  higher res" reading and is exactly why the live test matters: on
  this account, 480p video hit the paywall, not a resolution error.

## Why the live video test failed (and what it does NOT prove)

2026-10-27 live test: Video, 480p, 6s, 2:3, audio off, prompt "a quiet
golden-hour porch, camera slowly pushes in, no people" → subscription
dialog "Unlock your creativity with Imagine"
(`upsell.imagine.title-v4`), plans Plus $8/month and Premium $15/month.
No video, no asset ID, no quota wording.

That proves: on this Ruby Hart account, the web UI gates video behind
a subscription *before* any `conversations/new` video call completes.
It does NOT prove the video protocol is broken, and it does NOT prove
grok2api's Web video path is broken — grok2api's Web video is a
different caller (SSO session + egress lease, not the browser UI), and
its text-to-video path has not been live-tested in this thread.
Console video (`grok-imagine-video-1.5`, Super/paid Build, also
"including Free Console accounts" in the README's Console table) is a
separate provider path and was not tested either.

## What grok2api Web video does NOT support (by code)

- Image-to-video, reference-to-video, and video extension: `GenerateVideo`
  in `video.go` explicitly rejects any `ImageURL` or `ReferenceURLs`
  with "Grok Web currently only supports text-to-video; for image video
  use Build or Console Provider". So the three non-text kinds above
  exist in the web client but have no Web-provider implementation here.
- Audio assets, first/last frame, keyframes, GIF source, extension
  start time: no fields in `VideoRequest` handling for Web.
- Post-generation transforms are separate bundle routes, not video
  generation: `/rest/media/video/caption`, `/rest/media/video/music`,
  `/rest/media/video/upscale`, `/rest/media/video-sequence/create`.
  None is implemented in the Web provider (searched, non-test Go).

## What would prove video end-to-end (not yet approved)

1. A grok2api Web text-to-video call (SSO account, 6s, 720p, text-only)
   — would test the protocol without the browser UI paywall. Needs a
   live-call approval; it may still fail on quota/tier, and that
   failure would be informative, not a retry loop.
2. A paid subscription decision (Plus $8 / Premium $15 as shown) —
   would unblock the browser UI video path. Spend, not approved.
3. Image/reference/extension video — needs a new provider implementation
   (Build/Console first, per grok2api's own error message), not a
   bundle re-read. The bundle already gives the exact payload shapes
   above, so that implementation is spec-ready but unbuilt and
   untested.

## Sources

- grok2api fork: `backend/internal/infra/provider/web/video.go`
  (`GenerateVideo`, `videoCreatePayload`, `parseVideoStream`,
  `videoSegments`, `applyFreeWebVideoDurationCap`),
  `backend/internal/domain/settings/settings.go` (free cap 6),
  `backend/internal/infra/provider/web/video_duration_cap_test.go`.
- Local bundle: `~/workspace/grok-capture/bundle/1aw97vh8d1x5z.js`
  (media-store-v2: `generateVideoForImage`, the four `mediaGenInput`
  builders, `videoDurationOptions`, `allowedVideoResolution`).
- Live UI controls + paywall: 2026-10-27 browser test, Ruby Hart
  account, drafts only, nothing subscribed.
