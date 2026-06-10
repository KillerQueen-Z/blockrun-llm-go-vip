// Seedance video through BlockRun, paid per call in USDC (x402) on Base.
//
// Three modes:
//   - plain text-to-video
//   - real, specific person (RealFace: enroll once with on-phone consent)
//   - AI character / mascot (Virtual Portrait: single enroll, no liveness)
//
// Run with a funded Base wallet on the gateway:
//
//	export BLOCKRUN_WALLET_KEY=0x...   # or ~/.blockrun/.session
//	go run ./examples/seedance
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	vip "github.com/BlockRunAI/blockrun-llm-go-vip"
)

func main() {
	ctx := context.Background()

	// ---- 1. Plain text-to-video (Seedance), async submit→poll ----
	video, err := vip.NewVideo()
	if err != nil {
		log.Fatal(err)
	}
	dur := 5

	// Submit signs the x402 payment authorization and returns immediately;
	// generation runs asynchronously on the gateway. USDC settles only on the
	// first poll that observes completed — failed/abandoned jobs are never charged.
	job, err := video.Submit(ctx, "a neon-lit cyberpunk street, slow dolly forward", &vip.VideoGenerateOptions{
		Model:           "bytedance/seedance-2.0-fast",
		DurationSeconds: dur,
		Resolution:      "720p",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("submitted:", job.ID, "status:", job.Status)

	// Poll yourself...
	for !job.Done() {
		time.Sleep(12 * time.Second)
		if err := video.Poll(ctx, job); err != nil {
			log.Fatal(err)
		}
		fmt.Println("status:", job.Status)
	}
	// ...or just call video.Wait(ctx, job) to block until done.
	if job.Response != nil && len(job.Response.Data) > 0 {
		fmt.Println("video:", job.Response.Data[0].URL) // permanent BlockRun-hosted MP4
	}

	// ---- 2. AI character / mascot via Virtual Portrait (no liveness, $0.01) ----
	portrait, err := vip.NewPortrait()
	if err != nil {
		log.Fatal(err)
	}
	asset, err := portrait.Enroll(ctx, "Mascot", "https://example.com/character.jpg")
	if err != nil {
		log.Fatal(err)
	}
	job2, err := video.Generate(ctx, "the mascot waves and smiles in soft studio light", &vip.VideoGenerateOptions{
		Model:           "bytedance/seedance-2.0",
		RealFaceAssetID: asset.AssetID, // ta_xxxx — identity-consistent
		DurationSeconds: dur,
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(job2.Data) > 0 {
		fmt.Println("mascot video:", job2.Data[0].URL)
	}

	// ---- 3. Real, specific person via RealFace (one-time on-phone consent) ----
	// rf, _ := vip.NewRealFace()
	// started, _ := rf.Init(ctx, "Spokesperson — Q3", "")
	// fmt.Println("open on the rights-holder's phone:", started.H5Link)
	// rf.WaitForActive(ctx, started.GroupID, nil)            // after they nod + blink
	// person, _ := rf.Enroll(ctx, "Spokesperson — Q3", "https://example.com/person.jpg", started.GroupID)
	// job3, _ := video.Generate(ctx, "she waves warmly at the camera", &vip.VideoGenerateOptions{
	//     Model:           "bytedance/seedance-2.0",
	//     RealFaceAssetID: person.AssetID,
	// })
}
