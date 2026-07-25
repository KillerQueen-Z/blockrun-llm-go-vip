package vip

import "testing"

func TestModelCatalogContainsLiveSnapshot(t *testing.T) {
	models := Models()
	if got, want := len(models), 82; got != want {
		t.Fatalf("Models() returned %d models, want %d", got, want)
	}

	opus, ok := FindModel("anthropic/claude-opus-5")
	if !ok {
		t.Fatal("Claude Opus 5 is missing from the catalog")
	}
	if opus.Pricing.InputPerMillion != 5 || opus.Pricing.OutputPerMillion != 25 {
		t.Fatalf("unexpected Claude Opus 5 pricing: %+v", opus.Pricing)
	}
}

func TestModelsReturnsDefensiveCopies(t *testing.T) {
	models := Models()
	models[0].Categories[0] = "mutated"
	if Models()[0].Categories[0] == "mutated" {
		t.Fatal("Models() exposed the catalog's mutable categories")
	}
}
