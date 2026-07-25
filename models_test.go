package vip

import (
	"strings"
	"testing"
)

func TestModelCatalogContainsLiveSnapshot(t *testing.T) {
	models := Models()
	if len(models) == 0 {
		t.Fatal("Models() returned an empty catalog")
	}

	opus, ok := FindModel("anthropic/claude-opus-5")
	if !ok {
		t.Fatal("Claude Opus 5 is missing from the catalog")
	}
	if opus.Pricing.InputPerMillion != 5 || opus.Pricing.OutputPerMillion != 25 {
		t.Fatalf("unexpected Claude Opus 5 pricing: %+v", opus.Pricing)
	}
	if opus.BillingMode != "paid" {
		t.Fatalf("unexpected Claude Opus 5 billing mode: %q", opus.BillingMode)
	}

	for _, id := range []string{"openai/gpt-5.6-sol", "bytedance/seedance-2.0", "elevenlabs/v3"} {
		if _, ok := FindModel(id); !ok {
			t.Errorf("%s is missing from the catalog", id)
		}
	}
}

func TestCatalogEntriesAreWellFormed(t *testing.T) {
	seen := make(map[string]string)
	for _, model := range Models() {
		if model.Name == "" || model.Provider == "" || len(model.Categories) == 0 {
			t.Errorf("%s: incomplete entry %+v", model.ID, model)
		}
		if model.BillingMode == "" {
			t.Errorf("%s: missing billing mode", model.ID)
		}
		if !strings.Contains(model.ID, "/") {
			t.Errorf("%s: catalog ids are namespaced as provider/model", model.ID)
		}
		if prev, dup := seen[model.NativeID()]; dup {
			t.Errorf("%s and %s share the native id %q, so bare FindModel lookups are ambiguous",
				prev, model.ID, model.NativeID())
		}
		seen[model.NativeID()] = model.ID
	}
}

// The native passthrough clients take the bare provider id, so every catalog
// entry must be reachable by the same string a caller hands to Messages.New or
// Chat.Completions.New.
func TestFindModelAcceptsNativeIDs(t *testing.T) {
	opus, ok := FindModel("claude-opus-5")
	if !ok {
		t.Fatal("FindModel did not resolve the native id claude-opus-5")
	}
	if opus.ID != "anthropic/claude-opus-5" {
		t.Fatalf("resolved the wrong entry: %s", opus.ID)
	}
	if opus.NativeID() != "claude-opus-5" {
		t.Fatalf("unexpected native id: %s", opus.NativeID())
	}

	for _, model := range Models() {
		found, ok := FindModel(model.NativeID())
		if !ok {
			t.Errorf("%s: native id %q is unresolvable", model.ID, model.NativeID())
			continue
		}
		if found.ID != model.ID {
			t.Errorf("native id %q resolved to %s, want %s", model.NativeID(), found.ID, model.ID)
		}
	}

	if _, ok := FindModel("no-such-model"); ok {
		t.Error("FindModel matched an unknown id")
	}
}

func TestModelsReturnsDefensiveCopies(t *testing.T) {
	models := Models()
	models[0].Categories[0] = "mutated"
	if Models()[0].Categories[0] == "mutated" {
		t.Fatal("Models() exposed the catalog's mutable categories")
	}

	found, ok := FindModel(modelCatalog[0].ID)
	if !ok {
		t.Fatalf("%s is missing from the catalog", modelCatalog[0].ID)
	}
	found.Categories[0] = "mutated"
	if modelCatalog[0].Categories[0] == "mutated" {
		t.Fatal("FindModel exposed the catalog's mutable categories")
	}
}
