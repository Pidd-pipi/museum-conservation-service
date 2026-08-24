package main

type Artifact struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Material        string    `json:"material"`
	Humidity        float64   `json:"humidity"`
	HumiditySamples []float64 `json:"humidity_samples"`
	Status          string    `json:"status"`
	LastChecked     string    `json:"last_checked"`
}

// SamplesCopy returns an isolated copy of the humidity samples so mutations
// to the returned slice cannot affect the artifact's stored samples.
func (a Artifact) SamplesCopy() []float64 {
	if a.HumiditySamples == nil {
		return nil
	}
	out := make([]float64, len(a.HumiditySamples))
	copy(out, a.HumiditySamples)
	return out
}
