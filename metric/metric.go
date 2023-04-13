package metric

var m Metric = &emptyMetric{}

func SetMetric(injectMetric Metric) {
	m = injectMetric
}

// Count count 打点
func Count(name string, tags map[string]string, value float64) {
	m.Count(name, tags, value)
}

// Time Time 打点
func Time(name string, tags map[string]string, value float64) {
	m.Time(name, tags, value)
}

// Time Time 打点
func Summary(name string, tags map[string]string, value float64) {
	m.Summary(name, tags, value)
}

type Metric interface {
	// Count count 打点
	Count(name string, tags map[string]string, value float64)

	// Time Time 打点
	Time(name string, tags map[string]string, value float64)

	// Time Time 打点
	Summary(name string, tags map[string]string, value float64)
}

type emptyMetric struct {
}

// Count count 打点
func (em *emptyMetric) Count(name string, tags map[string]string, value float64) {
}

// Time Time 打点
func (em *emptyMetric) Time(name string, tags map[string]string, value float64) {
}

// Time Time 打点
func (em *emptyMetric) Summary(name string, tags map[string]string, value float64) {
}
