package hlims

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

var composeProjectPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func deploymentResponse(publicID, machineID, machineName, name, slug, directory, project, files string, notes sql.NullString) api.Deployment {
	result := api.Deployment{
		PublicId: publicID, MachinePublicId: machineID, MachineName: machineName,
		Name: name, Slug: slug, WorkingDirectory: directory, ComposeProject: project, Notes: stringPointer(notes),
	}
	_ = json.Unmarshal([]byte(files), &result.ComposeFiles)
	return result
}

func (s apiServer) ListDeployments(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListDeployments(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Deployment, 0, len(rows))
	for _, row := range rows {
		items = append(items, deploymentResponse(row.PublicID, row.MachinePublicID, row.MachineName, row.Name, row.Slug, row.WorkingDirectory, row.ComposeProject, row.ComposeFiles, row.Notes))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Deployment `json:"items"`
	}{items})
}

func (s apiServer) deploymentParams(r *http.Request, body api.DeploymentWrite, existingSlug string) (database.CreateDeploymentParams, error) {
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, existingSlug)
	if err != nil {
		return database.CreateDeploymentParams{}, err
	}
	machine, err := s.queries.GetMachineByPublicID(r.Context(), body.MachinePublicId)
	if err != nil {
		return database.CreateDeploymentParams{}, err
	}
	params := database.CreateDeploymentParams{MachineID: machine.ID, Name: name, Slug: slug, Notes: nullableTrimmed(body.Notes)}
	directory := strings.TrimSpace(body.WorkingDirectory)
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return params, errors.New("workingDirectory must be a clean absolute path on the deployment Machine")
	}
	project := strings.TrimSpace(body.ComposeProject)
	if !composeProjectPattern.MatchString(project) {
		return params, errors.New("composeProject must start with a lowercase letter or digit and contain only lowercase letters, digits, - or _")
	}
	if len(body.ComposeFiles) == 0 {
		return params, errors.New("composeFiles must contain at least one file")
	}
	paths := make([]string, 0, len(body.ComposeFiles))
	for _, path := range body.ComposeFiles {
		path = strings.TrimSpace(path)
		if path == "" || filepath.Clean(path) != path || path == "." || strings.HasPrefix(path, "../") || path == ".." {
			return params, errors.New("composeFiles paths must be clean nonempty paths")
		}
		paths = append(paths, path)
	}
	encoded, err := json.Marshal(paths)
	if err != nil {
		return params, err
	}
	params.WorkingDirectory = directory
	params.ComposeProject = project
	params.ComposeFiles = string(encoded)
	return params, nil
}

func (s apiServer) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.DeploymentWrite](w, r)
	if !ok {
		return
	}
	params, err := s.deploymentParams(r, body, "")
	if err != nil {
		s.writeParentOrInputError(w, err)
		return
	}
	params.ID, params.PublicID, err = newIDs()
	if err == nil {
		_, err = s.queries.CreateDeployment(r.Context(), params)
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeDeployment(w, r, params.PublicID, http.StatusCreated)
}

func (s apiServer) GetDeployment(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeDeployment(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeDeployment(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetDeploymentByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, deploymentResponse(row.PublicID, row.MachinePublicID, row.MachineName, row.Name, row.Slug, row.WorkingDirectory, row.ComposeProject, row.ComposeFiles, row.Notes))
}

func (s apiServer) UpdateDeployment(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.DeploymentWrite](w, r)
	if !ok {
		return
	}
	existing, err := s.queries.GetDeploymentByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	params, err := s.deploymentParams(r, body, existing.Slug)
	if err != nil {
		s.writeParentOrInputError(w, err)
		return
	}
	_, err = s.queries.UpdateDeployment(r.Context(), database.UpdateDeploymentParams{
		MachineID: params.MachineID, Name: params.Name, Slug: params.Slug,
		WorkingDirectory: params.WorkingDirectory, ComposeProject: params.ComposeProject,
		ComposeFiles: params.ComposeFiles,
		Notes:        params.Notes, PublicID: publicID,
	})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeDeployment(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteDeployment(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteDeployment(r.Context(), publicID)
	deleteResource(w, rows, err)
}
