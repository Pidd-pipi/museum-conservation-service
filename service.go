package main

type ConservationService struct {
	store *ArtifactStore
}

func NewConservationService(store *ArtifactStore) *ConservationService {
	return &ConservationService{store: store}
}

func (s *ConservationService) Artifacts() []Artifact { return s.store.List() }

func (s *ConservationService) ChangeStatus(id, status string) (Artifact, error) {
	if err := ValidateArtifactStatus(status); err != nil {
		return Artifact{}, err
	}
	return s.store.UpdateStatus(id, status)
}

type ArtifactHumidity struct {
	ID     string  `json:"id"`
	Latest float64 `json:"latest"`
}

// HumidityTrend returns the latest humidity reading of every artifact.
// A freshly allocated slice is returned on every call so that results do
// not alias one another across calls.
func (s *ConservationService) HumidityTrend() []ArtifactHumidity {
	items := s.store.List()
	trend := make([]ArtifactHumidity, 0, len(items))
	for _, item := range items {
		if len(item.HumiditySamples) > 0 {
			trend = append(trend, ArtifactHumidity{ID: item.ID, Latest: item.HumiditySamples[len(item.HumiditySamples)-1]})
		}
	}
	return trend
}
