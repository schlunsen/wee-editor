package sitegenerator

import (
	"testing"
	"time"

	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/stretchr/testify/assert"
)

// setupTestGenerator is defined in error_recovery_test.go and shared across all test files

// MockRepository is a mock implementation of the Repository interface for testing
type MockRepository struct {
	projects         map[string]*database.SiteProject
	steps            map[string][]*database.SiteStep
	artifacts        map[int64]*database.SiteArtifact
	templates        map[string]*database.AvailableTemplate
	projectArtifacts map[string][]*database.SiteArtifact
	createProjectErr error
	getProjectErr    error
	updateProjectErr error
	deleteProjectErr error
	listProjectsErr  error
	createStepErr    error
	updateStepErr    error
	getStepsErr      error
	saveArtifactErr  error
	getArtifactsErr  error
	getArtifactErr   error
	cacheTemplateErr error
	getTemplatesErr  error
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		projects:         make(map[string]*database.SiteProject),
		steps:            make(map[string][]*database.SiteStep),
		artifacts:        make(map[int64]*database.SiteArtifact),
		templates:        make(map[string]*database.AvailableTemplate),
		projectArtifacts: make(map[string][]*database.SiteArtifact),
	}
}

// CreateSiteProject implements the mock
func (m *MockRepository) CreateSiteProject(project *database.SiteProject) error {
	if m.createProjectErr != nil {
		return m.createProjectErr
	}
	m.projects[project.ID] = project
	m.steps[project.ID] = []*database.SiteStep{}
	m.projectArtifacts[project.ID] = []*database.SiteArtifact{}
	return nil
}

// GetSiteProject implements the mock
func (m *MockRepository) GetSiteProject(projectID string) (*database.SiteProject, error) {
	if m.getProjectErr != nil {
		return nil, m.getProjectErr
	}
	return m.projects[projectID], nil
}

// UpdateSiteProject implements the mock
func (m *MockRepository) UpdateSiteProject(project *database.SiteProject) error {
	if m.updateProjectErr != nil {
		return m.updateProjectErr
	}
	m.projects[project.ID] = project
	return nil
}

// DeleteSiteProject implements the mock
func (m *MockRepository) DeleteSiteProject(projectID string) error {
	if m.deleteProjectErr != nil {
		return m.deleteProjectErr
	}
	delete(m.projects, projectID)
	delete(m.steps, projectID)
	delete(m.projectArtifacts, projectID)
	return nil
}

// ListSiteProjects implements the mock
func (m *MockRepository) ListSiteProjects(limit, offset int) ([]*database.SiteProject, error) {
	if m.listProjectsErr != nil {
		return nil, m.listProjectsErr
	}
	var projects []*database.SiteProject
	for _, p := range m.projects {
		projects = append(projects, p)
	}
	return projects, nil
}

// CreateSiteStep implements the mock
func (m *MockRepository) CreateSiteStep(step *database.SiteStep) error {
	if m.createStepErr != nil {
		return m.createStepErr
	}
	m.steps[step.ProjectID] = append(m.steps[step.ProjectID], step)
	return nil
}

// UpdateSiteStep implements the mock
func (m *MockRepository) UpdateSiteStep(step *database.SiteStep) error {
	if m.updateStepErr != nil {
		return m.updateStepErr
	}
	for i, s := range m.steps[step.ProjectID] {
		if s.StepNumber == step.StepNumber {
			m.steps[step.ProjectID][i] = step
			return nil
		}
	}
	return nil
}

// GetSiteSteps implements the mock
func (m *MockRepository) GetSiteSteps(projectID string) ([]*database.SiteStep, error) {
	if m.getStepsErr != nil {
		return nil, m.getStepsErr
	}
	return m.steps[projectID], nil
}

// SaveSiteArtifact implements the mock
func (m *MockRepository) SaveSiteArtifact(artifact *database.SiteArtifact) error {
	if m.saveArtifactErr != nil {
		return m.saveArtifactErr
	}
	m.artifacts[artifact.ID] = artifact
	m.projectArtifacts[artifact.ProjectID] = append(m.projectArtifacts[artifact.ProjectID], artifact)
	return nil
}

// GetSiteArtifacts implements the mock
func (m *MockRepository) GetSiteArtifacts(projectID string) ([]*database.SiteArtifact, error) {
	if m.getArtifactsErr != nil {
		return nil, m.getArtifactsErr
	}
	return m.projectArtifacts[projectID], nil
}

// GetSiteArtifact implements the mock
func (m *MockRepository) GetSiteArtifact(artifactID int64) (*database.SiteArtifact, error) {
	if m.getArtifactErr != nil {
		return nil, m.getArtifactErr
	}
	return m.artifacts[artifactID], nil
}

// CacheLandingPageTemplate implements the mock
func (m *MockRepository) CacheLandingPageTemplate(template *database.AvailableTemplate) error {
	if m.cacheTemplateErr != nil {
		return m.cacheTemplateErr
	}
	m.templates[template.TemplateName] = template
	return nil
}

// GetLandingPageTemplates implements the mock
func (m *MockRepository) GetLandingPageTemplates(limit int) ([]*database.AvailableTemplate, error) {
	if m.getTemplatesErr != nil {
		return nil, m.getTemplatesErr
	}
	var templates []*database.AvailableTemplate
	for _, t := range m.templates {
		templates = append(templates, t)
	}
	return templates, nil
}

// TestNewSiteGenerator tests the constructor
func TestNewSiteGenerator(t *testing.T) {
	gen := setupTestGenerator(t)

	assert.NotNil(t, gen)
	assert.NotEmpty(t, gen.artifactStorePath)
	assert.Equal(t, MaxConcurrentProjects, gen.config.MaxConcurrentProjects)
}

// TestCreateProject tests project creation
func TestCreateProject(t *testing.T) {
	gen := setupTestGenerator(t)

	project, err := gen.CreateProject("A beautiful site")

	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.NotEmpty(t, project.ID)
	assert.Equal(t, "A beautiful site", project.UserDescription)
	assert.Equal(t, StatusPlanning, project.Status)
	assert.NotZero(t, project.CreatedAt)
}

// TestCreateProject_EmptyDescription tests project creation with empty description
func TestCreateProject_EmptyDescription(t *testing.T) {
	gen := setupTestGenerator(t)

	project, err := gen.CreateProject("")

	assert.Error(t, err)
	assert.Nil(t, project)
}

// TestCreateProject_WithEmptyReturns tests project creation with empty returns
func TestCreateProject_WithEmptyReturns(t *testing.T) {
	gen := setupTestGenerator(t)

	// Creating a project should succeed with a proper repository
	project, err := gen.CreateProject("Test project")

	// With proper repository, project should be created successfully
	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "Test project", project.UserDescription)
}

// TestGetProject tests retrieving a project
func TestGetProject(t *testing.T) {
	gen := setupTestGenerator(t)

	// Create a project first
	created, _ := gen.CreateProject("Test project")

	// Retrieve it
	project, err := gen.GetProject(created.ID)

	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, created.ID, project.ID)
	assert.Equal(t, "Test project", project.UserDescription)
}

// TestGetProject_InvalidID tests with invalid project ID
func TestGetProject_InvalidID(t *testing.T) {
	gen := setupTestGenerator(t)

	project, err := gen.GetProject("")

	assert.Error(t, err)
	assert.Nil(t, project)
}

// TestGetProject_NotFound tests with non-existent project ID
func TestGetProject_NotFound(t *testing.T) {
	gen := setupTestGenerator(t)

	project, err := gen.GetProject("nonexistent")

	assert.Error(t, err)
	assert.Nil(t, project)
}

// TestListProjects tests listing projects
func TestListProjects(t *testing.T) {
	gen := setupTestGenerator(t)

	// Create multiple projects
	gen.CreateProject("Project 1")
	gen.CreateProject("Project 2")
	gen.CreateProject("Project 3")

	projects, err := gen.ListProjects(10, 0)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(projects))
}

// TestListProjects_WithPagination tests pagination
func TestListProjects_WithPagination(t *testing.T) {
	gen := setupTestGenerator(t)

	// Create projects
	for i := 1; i <= 5; i++ {
		gen.CreateProject("Project " + string(rune(i)))
	}

	// Test with limit
	projects, err := gen.ListProjects(2, 0)

	assert.NoError(t, err)
	assert.NotNil(t, projects)
}

// TestDeleteProject tests project deletion
func TestDeleteProject(t *testing.T) {
	gen := setupTestGenerator(t)

	// Create and delete a project
	created, _ := gen.CreateProject("Test project")
	err := gen.DeleteProject(created.ID)

	assert.NoError(t, err)

	// Verify it's deleted
	_, err = gen.GetProject(created.ID)
	assert.Error(t, err)
}

// TestUpdateProjectStatus tests status updates
func TestUpdateProjectStatus(t *testing.T) {
	gen := setupTestGenerator(t)

	// Create a project
	created, _ := gen.CreateProject("Test project")

	// Update status
	err := gen.UpdateProjectStatus(created.ID, StatusRunning, "")

	assert.NoError(t, err)

	// Verify status was updated
	project, _ := gen.GetProject(created.ID)
	assert.Equal(t, StatusRunning, project.Status)
}

// TestUpdateProjectStatus_WithError tests status update with error message
func TestUpdateProjectStatus_WithError(t *testing.T) {
	gen := setupTestGenerator(t)

	created, _ := gen.CreateProject("Test project")
	errorMsg := "Generation failed"

	err := gen.UpdateProjectStatus(created.ID, StatusFailed, errorMsg)

	assert.NoError(t, err)

	project, _ := gen.GetProject(created.ID)
	assert.Equal(t, StatusFailed, project.Status)
	assert.Equal(t, errorMsg, project.ErrorMessage)
}

// TestCreateStep tests step creation
func TestCreateStep(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	step, err := gen.CreateStep(project.ID, SpecialistOrchestrator, 1)

	assert.NoError(t, err)
	assert.NotNil(t, step)
	assert.Equal(t, project.ID, step.ProjectID)
	assert.Equal(t, 1, step.StepNumber)
	assert.Equal(t, SpecialistOrchestrator, step.SpecialistType)
	assert.Equal(t, StepStatusPending, step.Status)
}

// TestUpdateStep tests step updates
func TestUpdateStep(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	step, _ := gen.CreateStep(project.ID, SpecialistOrchestrator, 1)

	// Update step status
	step.Status = StepStatusRunning
	err := gen.UpdateStep(step)

	assert.NoError(t, err)

	// Verify update
	steps, _ := gen.GetProjectSteps(project.ID)
	assert.Equal(t, StepStatusRunning, steps[0].Status)
}

// TestGetProjectSteps tests retrieving project steps
func TestGetProjectSteps(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	gen.CreateStep(project.ID, SpecialistOrchestrator, 1)
	gen.CreateStep(project.ID, SpecialistDesigner, 2)

	steps, err := gen.GetProjectSteps(project.ID)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(steps))
}

// TestSaveArtifact tests artifact saving
func TestSaveArtifact(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	step, _ := gen.CreateStep(project.ID, SpecialistOrchestrator, 1)

	artifact := &database.SiteArtifact{
		ProjectID:    project.ID,
		StepID:       step.ID,
		ArtifactType: "design_spec",
		Filename:     "design-spec.json",
		Content:      "Color: #FF0000",
	}

	err := gen.SaveArtifact(artifact)

	assert.NoError(t, err)
	assert.NotZero(t, artifact.CreatedAt)
}

// TestGetProjectArtifacts tests retrieving artifacts
func TestGetProjectArtifacts(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	step, _ := gen.CreateStep(project.ID, SpecialistOrchestrator, 1)

	artifact1 := &database.SiteArtifact{
		ProjectID:    project.ID,
		StepID:       step.ID,
		ArtifactType: "design_spec",
		Filename:     "design-spec.json",
		Content:      "Color: #FF0000",
	}
	step2, _ := gen.CreateStep(project.ID, SpecialistDesigner, 2)
	artifact2 := &database.SiteArtifact{
		ProjectID:    project.ID,
		StepID:       step2.ID,
		ArtifactType: "template",
		Filename:     "template.html",
		Content:      "<html></html>",
	}

	gen.SaveArtifact(artifact1)
	gen.SaveArtifact(artifact2)

	artifacts, err := gen.GetProjectArtifacts(project.ID)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(artifacts))
}

// TestGetArtifact tests retrieving a specific artifact
func TestGetArtifact(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	step, _ := gen.CreateStep(project.ID, SpecialistOrchestrator, 1)
	artifact := &database.SiteArtifact{
		ID:           1,
		ProjectID:    project.ID,
		StepID:       step.ID,
		ArtifactType: "design_spec",
		Filename:     "design-spec.json",
		Content:      "Color: #FF0000",
	}
	gen.SaveArtifact(artifact)

	retrieved, err := gen.GetArtifact(1)

	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "design_spec", retrieved.ArtifactType)
}

// TestGetProjectProgress tests progress calculation
func TestGetProjectProgress(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	step1, _ := gen.CreateStep(project.ID, SpecialistOrchestrator, 1)
	_, _ = gen.CreateStep(project.ID, SpecialistDesigner, 2)

	// Mark one step as completed
	step1.Status = StepStatusCompleted
	gen.UpdateStep(step1)

	progress, err := gen.GetProjectProgress(project.ID)

	assert.NoError(t, err)
	assert.NotNil(t, progress)
	assert.Equal(t, 2, progress["total_steps"])
	assert.Equal(t, 1, progress["completed_steps"])
	assert.Equal(t, 1, progress["pending_steps"])
}

// TestStoreOrchestrationPlan tests storing orchestration plan
func TestStoreOrchestrationPlan(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	plan := &ExecutionPlan{
		Steps: []PlanStep{
			{
				StepNumber:        1,
				SpecialistType:    SpecialistOrchestrator,
				Input:             "Plan the site structure",
				RequiredContext:   "User description",
				DeliverableFormat: "JSON",
			},
		},
	}

	err := gen.StoreOrchestrationPlan(project.ID, plan)

	assert.NoError(t, err)

	// Retrieve and verify
	retrieved, _ := gen.GetOrchestrationPlan(project.ID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, 1, len(retrieved.Steps))
}

// TestSetProjectCompletionTime tests setting completion time
func TestSetProjectCompletionTime(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	completionTime := time.Now()

	err := gen.SetProjectCompletionTime(project.ID, completionTime)

	assert.NoError(t, err)

	updated, _ := gen.GetProject(project.ID)
	assert.NotNil(t, updated.CompletedAt)
	assert.True(t, updated.CompletedAt.Equal(completionTime) || updated.CompletedAt.After(completionTime.Add(-time.Second)))
}

// TestGetProjectStep tests retrieving a specific step
func TestGetProjectStep(t *testing.T) {
	gen := setupTestGenerator(t)

	project, _ := gen.CreateProject("Test project")
	gen.CreateStep(project.ID, SpecialistOrchestrator, 1)
	gen.CreateStep(project.ID, SpecialistDesigner, 2)

	step, err := gen.GetProjectStep(project.ID, 2)

	assert.NoError(t, err)
	assert.NotNil(t, step)
	assert.Equal(t, 2, step.StepNumber)
	assert.Equal(t, SpecialistDesigner, step.SpecialistType)
}

// TestListProjects_Limits tests limit enforcement
func TestListProjects_Limits(t *testing.T) {
	gen := setupTestGenerator(t)

	// Create a project
	gen.CreateProject("Test")

	// Test with zero limit (should default to 50)
	projects, err := gen.ListProjects(0, 0)
	assert.NoError(t, err)
	assert.NotNil(t, projects)

	// Test with limit > 500 (should cap to 500)
	projects, err = gen.ListProjects(1000, 0)
	assert.NoError(t, err)
	assert.NotNil(t, projects)

	// Test with negative offset (should default to 0)
	projects, err = gen.ListProjects(10, -5)
	assert.NoError(t, err)
	assert.NotNil(t, projects)
}
