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

// SamplesCopy returns an isolated copy of the humidity samples.
func (a Artifact) SamplesCopy() []float64 {
	return a.HumiditySamples
}
