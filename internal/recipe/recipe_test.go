package recipe

import (
	"testing"

	"machine-simulator-2/internal/config"
)

func TestCatalogPopulateAndLookup(t *testing.T) {
	cat := NewCatalog()

	cycleTime := 4500
	scrapRate := 0.05
	templates := map[string]*config.LineTemplateConfig{
		"test-line": {
			Name: "test-line",
			Recipes: []config.RecipeConfig{
				{
					ProductID:       "PROD-A",
					Description:     "Product A",
					PlannedQtyRange: [2]int{50, 200},
					Overrides: map[int]config.RecipeStageOverrideConfig{
						1: {CycleTimeMS: &cycleTime, ScrapRate: &scrapRate},
					},
				},
				{
					ProductID:       "PROD-B",
					Description:     "Product B",
					PlannedQtyRange: [2]int{10, 50},
				},
			},
		},
	}

	cat.PopulateFromConfig(templates)

	recipes := cat.GetRecipes("test-line")
	if len(recipes) != 2 {
		t.Fatalf("expected 2 recipes, got %d", len(recipes))
	}

	r, err := cat.GetRecipe("test-line", "PROD-A")
	if err != nil {
		t.Fatal(err)
	}
	if r.PlannedQtyRange[0] != 50 || r.PlannedQtyRange[1] != 200 {
		t.Errorf("unexpected range: %v", r.PlannedQtyRange)
	}

	// Check overrides
	ov, ok := r.StageOverrides[1]
	if !ok {
		t.Fatal("expected stage 1 override")
	}
	if ov.CycleTimeMS == nil || *ov.CycleTimeMS != 4500 {
		t.Errorf("expected cycle time 4500, got %v", ov.CycleTimeMS)
	}

	// Not found
	_, err = cat.GetRecipe("test-line", "NONEXISTENT")
	if err == nil {
		t.Error("expected error for missing recipe")
	}

	_, err = cat.GetRecipe("nonexistent-template", "PROD-A")
	if err == nil {
		t.Error("expected error for missing template")
	}
}

func TestToMachineOverrides(t *testing.T) {
	ct := 3000
	sr := 0.1
	so := StageOverride{CycleTimeMS: &ct, ScrapRate: &sr}
	mo := so.ToMachineOverrides()
	if mo.CycleTimeMS == nil || *mo.CycleTimeMS != 3000 {
		t.Error("cycle time not transferred")
	}
	if mo.ScrapRate == nil || *mo.ScrapRate != 0.1 {
		t.Error("scrap rate not transferred")
	}
	if mo.BreakdownProbability != nil {
		t.Error("breakdown prob should be nil")
	}
}

func TestGetAllProducts(t *testing.T) {
	cat := NewCatalog()
	templates := map[string]*config.LineTemplateConfig{
		"line-a": {
			Recipes: []config.RecipeConfig{
				{ProductID: "PA1"}, {ProductID: "PA2"},
			},
		},
		"line-b": {
			Recipes: []config.RecipeConfig{
				{ProductID: "PB1"},
			},
		},
	}
	cat.PopulateFromConfig(templates)

	all := cat.GetAllProducts()
	if len(all["line-a"]) != 2 {
		t.Errorf("expected 2 products for line-a, got %d", len(all["line-a"]))
	}
	if len(all["line-b"]) != 1 {
		t.Errorf("expected 1 product for line-b, got %d", len(all["line-b"]))
	}
}
