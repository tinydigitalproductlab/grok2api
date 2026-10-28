package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chenyme/grok2api/backend/internal/domain/account"
	infraegress "github.com/chenyme/grok2api/backend/internal/infra/egress"
	"github.com/chenyme/grok2api/backend/internal/infra/provider"
	"github.com/chenyme/grok2api/backend/internal/infra/security"
)

const testPNGDataURI = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func TestGenerateVideoImageAndReferenceKinds(t *testing.T) {
	var lastPayload map[string]any
	uploadCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/http/upload-file-v2/direct":
			uploadCount++
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"fileMetadata":{"fileMetadataId":"meta_test"}}`)
		case "/rest/app-chat/conversations/new":
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode video payload: %v", err)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			lastPayload = payload
			writer.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(writer, `data: {"result":{"response":{"streamingVideoGenerationResponse":{"progress":100,"videoPostId":"post_1","videoUrl":"/videos/final.mp4"}}}}`+"\n\n")
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cipher, err := security.NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	encryptedToken, err := cipher.Encrypt("test-sso")
	if err != nil {
		t.Fatal(err)
	}
	adapter := NewAdapter(Config{
		BaseURL: server.URL, StatsigMode: "manual", StatsigManualValue: "test",
		VideoTimeoutSeconds: 5, FreeVideoDurationCap: 6,
	}, infraegress.NewManager(egressRepositoryStub{}, cipher), cipher, nil, nil)
	credential := account.Credential{
		ID: 1, Provider: account.ProviderWeb, WebTier: account.WebTierSuper,
		EncryptedAccessToken: encryptedToken,
	}

	// imageToVideo
	result, err := adapter.GenerateVideo(context.Background(), provider.VideoRequest{
		Credential: credential, Prompt: "pan left", Duration: 6,
		AspectRatio: "16:9", Resolution: "720p", ImageURL: testPNGDataURI,
	})
	if err != nil || result.URL == "" {
		t.Fatalf("image video result=%#v err=%v", result, err)
	}
	mediaGenInput, _ := lastPayload["mediaGenInput"].(map[string]any)
	imageToVideo, _ := mediaGenInput["imageToVideo"].(map[string]any)
	if imageToVideo == nil {
		t.Fatalf("image payload mediaGenInput=%v", mediaGenInput)
	}
	assets, _ := imageToVideo["inputAssets"].([]any)
	if len(assets) != 1 || assets[0] != "meta_test" {
		t.Fatalf("image inputAssets=%v", assets)
	}
	if imageToVideo["mode"] != "custom" {
		t.Fatalf("image mode=%v", imageToVideo["mode"])
	}
	if imageToVideo["useFirstFrame"] != true {
		t.Fatalf("image useFirstFrame=%v", imageToVideo["useFirstFrame"])
	}
	attachments, _ := lastPayload["imageAttachments"].([]any)
	if len(attachments) != 1 || attachments[0] != "meta_test" {
		t.Fatalf("image imageAttachments=%v", lastPayload["imageAttachments"])
	}

	// referenceToVideo
	uploadCount = 0
	result, err = adapter.GenerateVideo(context.Background(), provider.VideoRequest{
		Credential: credential, Prompt: "keep style", Duration: 6,
		AspectRatio: "16:9", Resolution: "720p",
		ReferenceURLs: []string{testPNGDataURI, testPNGDataURI},
	})
	if err != nil || result.URL == "" {
		t.Fatalf("reference video result=%#v err=%v", result, err)
	}
	if uploadCount != 2 {
		t.Fatalf("reference uploads=%d want 2", uploadCount)
	}
	mediaGenInput, _ = lastPayload["mediaGenInput"].(map[string]any)
	referenceToVideo, _ := mediaGenInput["referenceToVideo"].(map[string]any)
	if referenceToVideo == nil {
		t.Fatalf("reference payload mediaGenInput=%v", mediaGenInput)
	}
	if referenceToVideo["firstFrameAsset"] != "meta_test" {
		t.Fatalf("reference firstFrameAsset=%v", referenceToVideo["firstFrameAsset"])
	}
	refs, _ := referenceToVideo["mediaReferences"].([]any)
	if len(refs) != 2 {
		t.Fatalf("reference mediaReferences=%v", refs)
	}

	// reference audio stays rejected for Web (voice_id, not audio asset)
	if _, err := adapter.GenerateVideo(context.Background(), provider.VideoRequest{
		Credential: credential, Prompt: "x", Duration: 6, ReferenceAudios: []string{"voice_1"},
	}); err == nil {
		t.Fatal("reference audio for Web was accepted")
	}
	// video extension with a bare asset ID
	result, err = adapter.GenerateVideo(context.Background(), provider.VideoRequest{
		Credential: credential, Prompt: "continue", Duration: 6, VideoURL: "video_asset_1",
	})
	if err != nil || result.URL == "" {
		t.Fatalf("extension video result=%#v err=%v", result, err)
	}
	mediaGenInput, _ = lastPayload["mediaGenInput"].(map[string]any)
	videoExtension, _ := mediaGenInput["videoExtension"].(map[string]any)
	if videoExtension == nil {
		t.Fatalf("extension payload mediaGenInput=%v", mediaGenInput)
	}
	extAssets, _ := videoExtension["inputAssets"].([]any)
	if len(extAssets) != 1 || extAssets[0] != "video_asset_1" {
		t.Fatalf("extension inputAssets=%v", extAssets)
	}
	// video edit (not extension) stays Console-only for Web
	if _, err := adapter.GenerateVideo(context.Background(), provider.VideoRequest{
		Credential: credential, Prompt: "x", Duration: 6, Operation: provider.VideoOperationEdit, VideoURL: "video_asset_1",
	}); err == nil {
		t.Fatal("video edit for Web was accepted")
	}
}
