package main

type Artifact struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Material    string  `json:"material"`
	Humidity    float64 `json:"humidity"`
	Status      string  `json:"status"`
	LastChecked string  `json:"last_checked"`
}
