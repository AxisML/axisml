package workspace

import "github.com/axisml/axisml/axisml-platform/backend/internal/server"

// defaultImages is the Platform-curated catalog of selectable workspace base
// images (the dev-environment picker). It is a static curated set today; a
// later revision can source it from configuration without changing the API
// contract.
func defaultImages() []server.WorkspaceImage {
	return []server.WorkspaceImage{
		{Ref: "quay.io/jupyter/base-notebook:latest", DisplayName: "JupyterLab", Description: "Minimal JupyterLab notebook environment.", Kind: "jupyter", DefaultPort: 8888, Public: true},
		{Ref: "quay.io/jupyter/scipy-notebook:latest", DisplayName: "JupyterLab (SciPy)", Description: "JupyterLab with the SciPy / PyData stack preinstalled.", Kind: "jupyter", DefaultPort: 8888, Public: true},
		{Ref: "codercom/code-server:latest", DisplayName: "VS Code Server", Description: "Browser-based Visual Studio Code (code-server).", Kind: "vscode", DefaultPort: 8080, Public: true},
		{Ref: "rocker/rstudio:latest", DisplayName: "RStudio Server", Description: "RStudio IDE for R.", Kind: "rstudio", DefaultPort: 8787, Public: true},
	}
}

// Images returns the workspace base-image catalog.
func (s *Service) Images() []server.WorkspaceImage {
	return defaultImages()
}
