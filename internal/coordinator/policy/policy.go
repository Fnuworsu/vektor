package policy

type Engine struct {
	threshold float64
}

func NewEngine(threshold float64) *Engine {
	return &Engine{
		threshold: threshold,
	}
}

func (e *Engine) ShouldPrefetch(probability float64) bool {
	return probability >= e.threshold
}

func (e *Engine) UpdateThreshold(t float64) {
	e.threshold = t
}
