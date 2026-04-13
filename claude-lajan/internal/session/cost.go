package session

// modelPrice holds per-million-token pricing in USD.
type modelPrice struct {
	InputPerMTok      float64
	OutputPerMTok     float64
	CacheWritePerMTok float64
	CacheReadPerMTok  float64
}

// modelPricing maps model identifiers to their pricing.
// Source: Anthropic pricing page (as of 2026).
var modelPricing = map[string]modelPrice{
	"claude-opus-4-6":           {15.00, 75.00, 18.75, 1.50},
	"claude-sonnet-4-6":         {3.00, 15.00, 3.75, 0.30},
	"claude-sonnet-4-5":         {3.00, 15.00, 3.75, 0.30},
	"claude-haiku-4-5":          {0.80, 4.00, 1.00, 0.08},
	"claude-haiku-4-5-20251001": {0.80, 4.00, 1.00, 0.08},
}

// defaultPrice is the fallback pricing (sonnet-class model).
var defaultPrice = modelPrice{3.00, 15.00, 3.75, 0.30}

// CalculateCost returns the estimated USD cost for the given token counts and model.
func CalculateCost(model string, inputTokens, outputTokens, cacheCreate, cacheRead int64) float64 {
	p, ok := modelPricing[model]
	if !ok {
		p = defaultPrice
	}
	const mTok = 1_000_000.0
	return (float64(inputTokens)*p.InputPerMTok +
		float64(outputTokens)*p.OutputPerMTok +
		float64(cacheCreate)*p.CacheWritePerMTok +
		float64(cacheRead)*p.CacheReadPerMTok) / mTok
}
