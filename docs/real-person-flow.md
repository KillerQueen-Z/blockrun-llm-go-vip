# Real-Person Video — the supported flow (Go)

**Yes, real people are supported.** A specific, real human can appear consistently
across as many Seedance videos as you want. This document is the canonical end-to-end
flow for doing that with `blockrun-llm-go-vip`, plus the decision tree, state machine,
and error states.

It exists because the capability is easy to miss: if you try to upload a real-person
photo directly to a generic image-to-video call (or to Sora), it gets **rejected** — and
it's natural to conclude "the API doesn't support real people." It does. Real-person
video just goes through a different, consent-based door: **RealFace enrollment**, not a
raw face upload.

> **Licensing:** BlockRun holds ByteDance's (advanced) creation-rights package for
> Seedance 2.0, which is what licenses subject upload (`real_face_asset_id`). The
> capability is live today at $0.01/enrollment; customers do not owe a separate package
> fee (confirm exact contract terms with your BlockRun contact).

---

## TL;DR

```go
import (
    "context"
    vip "github.com/BlockRunAI/blockrun-llm-go-vip"
)

ctx := context.Background()

rf, _ := vip.NewRealFace()
started, _ := rf.Init(ctx, "Spokesperson — Q3 campaign", "")   // FREE
fmt.Println("Open on the rights-holder's phone:", started.H5Link) // QR / mobile link

rf.WaitForActive(ctx, started.GroupID, nil)                    // after they nod + blink (~1 min)
asset, _ := rf.Enroll(ctx,                                     // PAID — $0.01 USDC, one-time
    "Spokesperson — Q3 campaign",
    "https://example.com/person.jpg",
    started.GroupID,
)

video, _ := vip.NewVideo()
job, _ := video.Generate(ctx, "she smiles warmly and waves at the camera in soft studio light",
    &vip.VideoGenerateOptions{
        Model:           "bytedance/seedance-2.0",
        RealFaceAssetID: asset.AssetID,                        // ta_xxxx — the real person
    })
fmt.Println(job.Data[0].URL)                                  // permanent MP4 of that person
```

That `ta_xxxx` is reusable forever — pay the per-clip cost each time, never re-enroll.
For long generations use the async form: `video.Submit(...)` → `video.Poll(ctx, job)` /
`video.Wait(ctx, job)`.

---

## Common misunderstanding → correction

| What people believe | What's actually true |
|---|---|
| "The API doesn't support real people, only virtual portraits." | Real people **are** supported via **RealFace**. Virtual Portrait is the *separate* path for AI-generated characters. |
| "blockrun doesn't allow real-person likeness." | It does — gated behind a lightweight **on-phone liveness check** (proof of consent), **not** KYC and **not** an offline rights-holder contract. |
| "I uploaded a face and it was rejected, so it's unsupported." | A **raw face upload** to Sora / generic image-to-video is blocked by design (anti-deepfake). The supported route is **enroll once via RealFace, then reference the `ta_xxxx`** — not a per-call face upload. |
| "Real-person needs a 10万/30万 rights package / offline contract first." | The technical capability is **live today** at $0.01/enrollment. BlockRun already holds the creation-rights package, so there is no extra per-customer package fee; the only consent mechanism the API requires is the liveness check. |

---

## Two asset types — which door do you use?

Both produce a `ta_xxxx` asset that you pass as `RealFaceAssetID` on Seedance
2.0 / 2.0-fast. The only difference is what the asset represents and whether consent
liveness is required.

```
                Is the subject a REAL, specific human?
                          │
            ┌─────────────┴─────────────┐
           YES                          NO  (AI character / mascot / avatar)
            │                            │
       ┌────▼─────┐                ┌─────▼──────────┐
       │ RealFace │                │    Portrait     │
       └────┬─────┘                └─────┬──────────┘
   consent liveness (~1 min,        no liveness, single
   nod+blink on phone, NO KYC)      $0.01 x402 call
            │                            │
            └──────────────┬─────────────┘
                       ta_xxxx
                           │
                 RealFaceAssetID on
              Seedance 2.0 / 2.0-fast video
```

| | `vip.Portrait` | `vip.RealFace` |
|---|---|---|
| Subject | AI-generated character | A real, specific person |
| Liveness check | None | Required (~1 min on the rights-holder's phone) |
| KYC / government ID | No | No |
| Upstream verification | None | Biometric match: enrolled photo ↔ live H5 face |
| Price | $0.01 USDC, one-time | $0.01 USDC, one-time |
| Compatible models | Seedance 2.0 / 2.0-fast | Seedance 2.0 / 2.0-fast |
| Constructor | `vip.NewPortrait()` | `vip.NewRealFace()` |

> **Why no raw face upload?** Sora 2 (and generic image-to-video) reject reference
> images containing recognizable human faces — a deliberate anti-deepfake guard. So the
> only consented real-person path is the enroll-once-then-reference model below. The
> client enforces the same shape: `RealFaceAssetID` and `ImageURL` are **mutually
> exclusive**, and `RealFaceAssetID` is only accepted on Seedance 2.0 / 2.0-fast.

---

## RealFace state machine

```
   ┌──────────────────────────── rf.Init(ctx, name, "") ── FREE
   │                              returns { GroupID, H5Link, ExpiresInSeconds: 120 }
   ▼
[ pending_validation ] ── rights-holder opens H5Link on their phone ───┐
   │   ▲                                                               │
   │   │  (120s H5 session expired?                                    │
   │   │   rf.Init(ctx, name, groupID) to refresh)                     │
   │   └───────────────────────────────────────────────────────────┐  │
   │                                                                │  │
   │   poll rf.Status(ctx, groupID)  ── FREE (every 3–5s)           │  │
   │   or rf.WaitForActive(ctx, groupID, nil)                       │  │
   ▼                                                                │  │
[ active / ready_to_finalize ] ◄────── nod + blink, 2–4s liveness ─┘◄─┘
   │
   │   rf.Enroll(ctx, name, imageURL, groupID) ── PAID $0.01 USDC (x402)
   │   upstream face-matches imageURL against the live H5 face
   ▼
[ enrolled ]  →  asset.AssetID = "ta_xxxx"
   │
   ▼
video.Generate(ctx, prompt, &vip.VideoGenerateOptions{
    Model: "bytedance/seedance-2.0", RealFaceAssetID: "ta_xxxx"})   ← reuse forever
```

| Step | Call | Cost | Notes |
|---|---|---|---|
| 1. Init | `rf.Init(ctx, name, "")` | FREE | Returns `GroupID` + `H5Link`. Rate-limited (10/hr/IP). H5 session lives 120s. |
| 2. Liveness | *(rights-holder, on phone)* | — | Scan `H5Link` as a QR or open on mobile. Nod + blink, 2–4s. No login, no ID upload. Camera-blocked fallback: record + upload a short video. Works on a laptop webcam too. |
| 3. Poll | `rf.Status(ctx, groupID)` / `rf.WaitForActive(ctx, groupID, nil)` | FREE | `pending_validation` → `active`. Poll every 3–5s. |
| 4. Enroll | `rf.Enroll(ctx, name, imageURL, groupID)` | **$0.01 USDC** | Settles **only** after the face-match succeeds. Returns `ta_xxxx`. |
| 5. Generate | `video.Generate(ctx, …, &vip.VideoGenerateOptions{RealFaceAssetID: …})` | per-clip | Reuse the `ta_xxxx` across unlimited clips. |
| —. List | `rf.ListRealFaces(ctx, walletAddr)` | FREE | All RealFace assets this wallet has enrolled. |

Whole sequence typically completes in **under 3 minutes** — mostly the person finding
their phone and tapping through the H5.

`WaitForActive` accepts `*vip.WaitForActiveOptions{Timeout, PollInterval}` (defaults:
180s budget, 4s interval; keep the interval ≥3s for rate limits).

---

## On-phone verification — what the rights-holder does

This is step 2 above — the part that happens on the **real person's own phone**. It takes
~1 minute and requires **no login, no password, no email, no ID upload, and no
personal-info entry**. Hand this section to the person being enrolled.

`rf.Init(...)` returns an `H5Link` (a `kyc.byteintl.com` URL — BlockRun's identity-
verification partner). Get it onto the rights-holder's phone via a QR code (render
`H5Link` as a QR; iOS/Android cameras auto-detect it) or by sending the link.

| # | On their phone |
|---|---|
| 1 | They open the link → an H5 page loads. URL is `kyc.byteintl.com` |
| 2 | Browser asks **"Allow camera access?"** → they tap **Allow** |
| 3 | A circle with a live camera feed appears → they position their face inside it |
| 4 | The page prompts **"please nod"**, then **"please blink"** → each ~1–2s. Total recording **2–4 seconds** |
| 5 | Page shows **"Verification completed. You can close this page now."** |

Total time **~60 seconds**. If the camera is denied, the H5 offers an alternate method:
record a 2–4s nod+blink clip with the native camera and upload it. The `H5Link` also
works in any laptop browser with a webcam.

**The 120-second window:** the H5 session token expires 120s after `Init`. If they don't
finish in time they'll see **"Session expired"** — refresh with the same group and have
them re-scan:

```go
rf.Init(ctx, "Spokesperson — Q3 campaign", started.GroupID) // fresh H5Link, same group
```

**Privacy:** the live face video goes **directly from their phone to the upstream identity
service** (`kyc.byteintl.com`); **BlockRun servers never receive the face video or any
biometric data**. The upstream service only keeps enough to perform a one-time face-match
against the photo you pass to `Enroll`. After the match, your supplied photo is the asset
of record — the live video isn't referenced again in later Seedance generations.

---

## Error / state reference

The single x402 transport handles the money: the free `Init` / `Status` /
`ListRealFaces` calls return `200` directly (nothing signed), and only `Enroll` triggers
the $0.01 settlement.

| Condition | HTTP | Surface | Did payment settle? |
|---|---|---|---|
| Group not yet `active` (liveness not done) | 425 | error from `Enroll` | **No** |
| Face-match failed (photo ≠ live face) | 422 | error from `Enroll` | **No** |
| Upload to inference partner failed | 502 | error from `Enroll` | **No** |
| Bad request body / image URL / group id | 400 | error (pre-payment) | — |
| Rate limit on `Init` / `Status` | 429 | error | — |
| Group never reaches `active` in time | — | error from `WaitForActive` | — |
| `ImageURL` **and** `RealFaceAssetID` both set | (client) | error from `video.Submit`/`Generate` | — |
| `RealFaceAssetID` not starting with `ta_` | (client) | error from `video.Submit`/`Generate` | — |
| Enrolled and active | 200 | `Enroll` returns `RealFaceEnrollment{AssetID: "ta_…"}` | **Yes** |

`Enroll` settlement is consent-safe: if the live face doesn't match the photo (422) or
the rights-holder hasn't finished the liveness check (425), **no USDC moves**.

---

## AI character instead of a real person

If the subject is a synthetic character (mascot, avatar, virtual spokesperson), skip the
liveness step entirely — enroll a **Portrait** and use the identical `ta_xxxx` →
`RealFaceAssetID` flow:

```go
p, _ := vip.NewPortrait()
asset, _ := p.Enroll(ctx, "Mascot", "https://example.com/character.jpg") // $0.01
// pass asset.AssetID as RealFaceAssetID on Seedance 2.0 / 2.0-fast
```

---

## See also

- Package usage: [`../README.md`](../README.md)
- Runnable example: [`../examples/seedance`](../examples/seedance)
- Python counterpart: `blockrun-llm-vip` `docs/real-person-flow.md`
- RealFace Studio (no-code web flow): https://blockrun.ai/studio/realface
