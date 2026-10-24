package recipe

import (
	"fmt"
	"sync"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

// Recipe represents a product recipe for a line type.
type Recipe struct {
	ProductID       string
	Description     string
	PlannedQtyRange [2]int
	StageOverrides  map[int]StageOverride
}

// StageOverride holds per-stage parameter overrides for a recipe.
type StageOverride struct {
	CycleTimeMS          *int
	ScrapRate            *float64
	BreakdownProbability *float64
}

// Catalog holds all recipes organized by line template.
type Catalog struct {
	mu      sync.RWMutex
	recipes map[string][]Recipe // keyed by line template name
}

// NewCatalog creates an empty recipe catalog.
func NewCatalog() *Catalog {
	return &Catalog{
		recipes: make(map[string][]Recipe),
	}
}

// PopulateFromConfig loads recipes from all line templates in the config.
func (c *Catalog) PopulateFromConfig(templates map[string]*config.LineTemplateConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for name, tmpl := range templates {
		for _, r := range tmpl.Recipes {
			recipe := Recipe{
				ProductID:       r.ProductID,
				Description:     r.Description,
				PlannedQtyRange: r.PlannedQtyRange,
				StageOverrides:  make(map[int]StageOverride),
			}
			for stage, ov := range r.Overrides {
				recipe.StageOverrides[stage] = StageOverride{
					CycleTimeMS:          ov.CycleTimeMS,
					ScrapRate:            ov.ScrapRate,
					BreakdownProbability: ov.BreakdownProbability,
				}
			}
			c.recipes[name] = append(c.recipes[name], recipe)
		}
	}
}

// GetRecipes returns all recipes for a line template.
func (c *Catalog) GetRecipes(templateName string) []Recipe {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.recipes[templateName]
}

// GetRecipe returns a specific recipe by template and product ID.
func (c *Catalog) GetRecipe(templateName, productID string) (*Recipe, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, r := range c.recipes[templateName] {
		if r.ProductID == productID {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("recipe %q not found in template %q", productID, templateName)
}

// GetAllProducts returns all product IDs across all templates.
func (c *Catalog) GetAllProducts() map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string][]string)
	for tmpl, recipes := range c.recipes {
		for _, r := range recipes {
			result[tmpl] = append(result[tmpl], r.ProductID)
		}
	}
	return result
}

// ToMachineOverrides converts a recipe's stage override to machine.RecipeOverrides.
func (so StageOverride) ToMachineOverrides() machine.RecipeOverrides {
	return machine.RecipeOverrides{
		CycleTimeMS:          so.CycleTimeMS,
		ScrapRate:            so.ScrapRate,
		BreakdownProbability: so.BreakdownProbability,
	}
}

// StageOverrides returns a product's machine overrides keyed by stage number.
// It lets the line manager apply a recipe when a job is loaded.
func (c *Catalog) StageOverrides(templateName, productID string) (map[int]machine.RecipeOverrides, error) {
	r, err := c.GetRecipe(templateName, productID)
	if err != nil {
		return nil, err
	}
	out := make(map[int]machine.RecipeOverrides, len(r.StageOverrides))
	for stage, so := range r.StageOverrides {
		out[stage] = so.ToMachineOverrides()
	}
	return out, nil
}
