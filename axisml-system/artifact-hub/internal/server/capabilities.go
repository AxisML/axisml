package server

// Capabilities describes what Artifact Hub supports in the current deployment
// form. Both forms back the registry with an OCI store (Standard: zot; Lite: a
// local zot/in-process store), so the artifact surface is identical across forms.
type Capabilities struct {
	// Kinds are the artifact kinds served (model / image / dataset).
	Kinds []string `json:"kinds" desc:"Artifact kinds served in this deployment form (model, dataset, image)."`
	// Upload reports whether two-phase artifact upload is available.
	Upload bool `json:"upload" desc:"Whether two-phase artifact upload is available in this deployment form."`
}
