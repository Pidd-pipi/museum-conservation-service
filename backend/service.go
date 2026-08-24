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
