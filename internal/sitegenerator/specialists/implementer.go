// Package specialists implements the Implementation specialist
package specialists

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ImplementationSpecialist creates Nuxt UI projects and generates static sites
type ImplementationSpecialist struct {
	BaseSpecialist
	workspaceDir string
}

// NewImplementationSpecialist creates a new implementation specialist
func NewImplementationSpecialist(workspaceDir string) *ImplementationSpecialist {
	return &ImplementationSpecialist{
		BaseSpecialist: BaseSpecialist{
			Name:             "Implementation Specialist",
			Type:             "implementer",
			Timeout:          600 * time.Second, // Increased for npm operations
			MaxRetries:       3,
			RetryDelay:       2 * time.Second,
			OutputValidation: true,
		},
		workspaceDir: workspaceDir,
	}
}

// Execute runs the implementation process
func (is *ImplementationSpecialist) Execute(
	ctx context.Context,
	handover *HandoverContext,
) (interface{}, error) {
	// Validate input
	if err := is.ValidateInput(handover); err != nil {
		return nil, err
	}

	// Create a context with timeout
	execCtx, cancel := context.WithTimeout(ctx, is.Timeout)
	defer cancel()

	// Execute with timeout
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := is.executeImplementation(execCtx, handover)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-execCtx.Done():
		return nil, fmt.Errorf("implementation specialist timeout")
	}
}

// executeImplementation performs the actual implementation logic
func (is *ImplementationSpecialist) executeImplementation(
	ctx context.Context,
	handover *HandoverContext,
) (*ImplementationOutput, error) {
	// Extract design info from previous step
	var designInfo *DesignOutput
	if handover.PreviousStepOutput != nil {
		if data, ok := handover.PreviousStepOutput.(*DesignOutput); ok {
			designInfo = data
		}
	}

	if designInfo == nil {
		return nil, fmt.Errorf("design output is required for implementation")
	}

	// Get user description
	userDescription := handover.UserDescription
	if userDescription == "" {
		return nil, fmt.Errorf("user description is required for implementation")
	}

	// Create Nuxt project
	output, err := is.createNuxtProject(ctx, userDescription, designInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create Nuxt project: %w", err)
	}

	// Validate output
	if err := is.ValidateOutput(output); err != nil {
		return nil, fmt.Errorf("implementation specialist output validation failed: %w", err)
	}

	return output, nil
}

// createNuxtProject creates a complete Nuxt UI project
func (is *ImplementationSpecialist) createNuxtProject(
	ctx context.Context,
	userDescription string,
	designInfo *DesignOutput,
) (*ImplementationOutput, error) {
	buildLog := strings.Builder{}
	projectName := fmt.Sprintf("nuxt-site-%d", time.Now().Unix())
	projectPath := filepath.Join(is.workspaceDir, projectName)

	// Ensure workspace directory exists
	if err := os.MkdirAll(is.workspaceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	buildLog.WriteString(fmt.Sprintf("Creating Nuxt project at: %s\n", projectPath))

	// Step 1: Create Nuxt project with UI template
	buildLog.WriteString("\n=== Step 1: Creating Nuxt UI project ===\n")
	if err := is.runCommand(ctx, is.workspaceDir, &buildLog,
		"npx", "nuxi@latest", "init", projectName, "-t", "ui"); err != nil {
		return is.buildErrorOutput(projectPath, buildLog.String(), err)
	}

	// Step 2: Install dependencies
	buildLog.WriteString("\n=== Step 2: Installing dependencies ===\n")
	if err := is.runCommand(ctx, projectPath, &buildLog, "npm", "install"); err != nil {
		return is.buildErrorOutput(projectPath, buildLog.String(), err)
	}

	// Step 3: Configure Nuxt for SPA mode
	buildLog.WriteString("\n=== Step 3: Configuring Nuxt ===\n")
	nuxtConfig := is.generateNuxtConfig(designInfo)
	if err := is.writeFile(filepath.Join(projectPath, "nuxt.config.ts"), nuxtConfig); err != nil {
		return is.buildErrorOutput(projectPath, buildLog.String(), err)
	}
	buildLog.WriteString("✓ nuxt.config.ts written\n")

	// Step 4: Create app.config.ts with theme
	buildLog.WriteString("\n=== Step 4: Applying theme ===\n")
	appConfig := designInfo.AppConfigTS
	if appConfig == "" {
		appConfig = is.generateAppConfig(designInfo)
	}
	if err := is.writeFile(filepath.Join(projectPath, "app", "app.config.ts"), appConfig); err != nil {
		return is.buildErrorOutput(projectPath, buildLog.String(), err)
	}
	buildLog.WriteString("✓ app.config.ts written\n")

	// Step 5: Create app.vue root component
	buildLog.WriteString("\n=== Step 5: Creating root component ===\n")
	appVue := is.generateAppVue()
	if err := is.writeFile(filepath.Join(projectPath, "app", "app.vue"), appVue); err != nil {
		return is.buildErrorOutput(projectPath, buildLog.String(), err)
	}
	buildLog.WriteString("✓ app.vue written\n")

	// Step 6: Create pages directory and pages
	buildLog.WriteString("\n=== Step 6: Creating pages ===\n")
	pagesDir := filepath.Join(projectPath, "app", "pages")
	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		return is.buildErrorOutput(projectPath, buildLog.String(), err)
	}

	pagesCreated := []string{}
	componentsUsed := make(map[string]bool)

	for _, pageDef := range designInfo.Pages {
		pageContent := is.generatePageVue(pageDef, userDescription)
		pagePath := filepath.Join(pagesDir, pageDef.Name+".vue")

		if err := is.writeFile(pagePath, pageContent); err != nil {
			return is.buildErrorOutput(projectPath, buildLog.String(), err)
		}

		pagesCreated = append(pagesCreated, pageDef.Name+".vue")
		buildLog.WriteString(fmt.Sprintf("✓ %s.vue created\n", pageDef.Name))

		// Track components used
		for _, comp := range pageDef.Components {
			componentsUsed[comp] = true
		}
	}

	// Step 7: Generate static site
	buildLog.WriteString("\n=== Step 7: Generating static site ===\n")
	if err := is.runCommand(ctx, projectPath, &buildLog, "npm", "run", "generate"); err != nil {
		return is.buildErrorOutput(projectPath, buildLog.String(), err)
	}

	// Step 8: Validate output
	buildLog.WriteString("\n=== Step 8: Validating output ===\n")
	generatedPath := filepath.Join(projectPath, ".output", "public")
	validationResult := is.validateBuildOutput(generatedPath, &buildLog)

	// Read generated files for output
	packageJSONContent, _ := os.ReadFile(filepath.Join(projectPath, "package.json"))

	// Build components list
	components := []string{}
	for comp := range componentsUsed {
		components = append(components, comp)
	}

	// Build successful output
	output := &ImplementationOutput{
		ProjectPath:       projectPath,
		GeneratedPath:     generatedPath,
		NuxtConfigTS:      nuxtConfig,
		AppConfigTS:       appConfig,
		PackageJSON:       string(packageJSONContent),
		PagesCreated:      pagesCreated,
		ComponentsUsed:    components,
		BuildLog:          buildLog.String(),
		ValidationResults: validationResult,
		StaticSiteReady:   validationResult.IsValid,
		CreatedAt:         time.Now(),
	}

	return output, nil
}

// runCommand executes a shell command and captures output
func (is *ImplementationSpecialist) runCommand(
	ctx context.Context,
	workDir string,
	log *strings.Builder,
	name string,
	args ...string,
) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir

	// Capture stdout and stderr
	output, err := cmd.CombinedOutput()
	log.WriteString(string(output))

	if err != nil {
		log.WriteString(fmt.Sprintf("\n❌ Command failed: %s %s\n", name, strings.Join(args, " ")))
		log.WriteString(fmt.Sprintf("Error: %v\n", err))
		return fmt.Errorf("command failed: %w", err)
	}

	log.WriteString(fmt.Sprintf("✓ Command succeeded: %s %s\n", name, strings.Join(args, " ")))
	return nil
}

// writeFile writes content to a file
func (is *ImplementationSpecialist) writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// generateNuxtConfig generates nuxt.config.ts content
func (is *ImplementationSpecialist) generateNuxtConfig(designInfo *DesignOutput) string {
	return `// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  extends: ['@nuxt/ui-pro'],
  modules: ['@nuxt/ui'],

  ssr: false,  // Disable server-side rendering (SPA mode)

  devtools: { enabled: true },

  compatibilityDate: '2025-01-27'
})
`
}

// generateAppConfig generates app.config.ts content
func (is *ImplementationSpecialist) generateAppConfig(designInfo *DesignOutput) string {
	// Extract primary color name from design
	primaryColor := "blue"
	if designInfo.NuxtUIConfig.Theme != "" {
		primaryColor = designInfo.NuxtUIConfig.Theme
	}

	return fmt.Sprintf(`export default defineAppConfig({
  ui: {
    primary: '%s',
    gray: 'slate',

    button: {
      rounded: 'rounded-lg',
      default: {
        size: 'md',
        color: 'primary',
        variant: 'solid'
      }
    },

    card: {
      rounded: 'rounded-xl',
      shadow: 'shadow-lg',
      ring: 'ring-1 ring-gray-200 dark:ring-gray-800'
    }
  }
})
`, primaryColor)
}

// generateAppVue generates app.vue content
func (is *ImplementationSpecialist) generateAppVue() string {
	return `<template>
  <div>
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </div>
</template>
`
}

// generatePageVue generates a Vue page from a PageDefinition
func (is *ImplementationSpecialist) generatePageVue(pageDef PageDefinition, userDescription string) string {
	var sb strings.Builder

	// Script setup section
	sb.WriteString("<script setup lang=\"ts\">\n")
	sb.WriteString(fmt.Sprintf(`useSeoMeta({
  title: '%s',
  description: '%s'
})
`, pageDef.Title, pageDef.MetaTags["description"]))

	// Add form state for contact pages
	if strings.Contains(strings.ToLower(pageDef.Name), "contact") {
		sb.WriteString(`
const state = reactive({
  name: '',
  email: '',
  message: ''
})

async function onSubmit() {
  console.log('Form submitted:', state)
  // Add your form submission logic here
}
`)
	}

	sb.WriteString("</script>\n\n")

	// Template section
	sb.WriteString("<template>\n  <div>\n")

	// Generate sections
	for _, section := range pageDef.Sections {
		sb.WriteString(is.generateSectionHTML(section, userDescription))
	}

	sb.WriteString("  </div>\n</template>\n")

	return sb.String()
}

// generateSectionHTML generates HTML for a section
func (is *ImplementationSpecialist) generateSectionHTML(section SectionDefinition, userDescription string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("    <!-- %s Section -->\n", section.Name))

	switch section.Type {
	case "hero":
		sb.WriteString(`    <UContainer class="py-24">
      <div class="text-center">
        <h1 class="text-4xl font-bold mb-4">
          Welcome to Our Site
        </h1>
        <p class="text-lg text-gray-600 dark:text-gray-400 mb-8 max-w-2xl mx-auto">
          ` + userDescription + `
        </p>
        <UButton size="lg" to="/features">
          Get Started
        </UButton>
      </div>
    </UContainer>

`)

	case "grid":
		sb.WriteString(`    <UContainer class="py-16">
      <h2 class="text-3xl font-bold mb-12 text-center">
        ` + section.Name + `
      </h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
        <UCard
          v-for="i in 3"
          :key="i"
          :ui="{ body: { padding: 'p-6' } }"
        >
          <h3 class="text-xl font-semibold mb-2">
            Feature {{ i }}
          </h3>
          <p class="text-gray-600 dark:text-gray-400">
            Description of feature {{ i }}
          </p>
        </UCard>
      </div>
    </UContainer>

`)

	case "form":
		sb.WriteString(`    <UContainer class="py-24">
      <div class="max-w-2xl mx-auto">
        <h1 class="text-4xl font-bold mb-4">
          Contact Us
        </h1>
        <p class="text-lg text-gray-600 dark:text-gray-400 mb-8">
          Have a question? We'd love to hear from you.
        </p>

        <UForm :state="state" @submit="onSubmit" class="space-y-4">
          <UFormGroup label="Name" name="name">
            <UInput v-model="state.name" />
          </UFormGroup>

          <UFormGroup label="Email" name="email">
            <UInput v-model="state.email" type="email" />
          </UFormGroup>

          <UFormGroup label="Message" name="message">
            <UTextarea v-model="state.message" :rows="5" />
          </UFormGroup>

          <UButton type="submit" block>
            Send Message
          </UButton>
        </UForm>
      </div>
    </UContainer>

`)

	case "cta":
		sb.WriteString(`    <UContainer class="py-16">
      <div class="bg-primary-50 dark:bg-primary-950 rounded-2xl p-12 text-center">
        <h2 class="text-3xl font-bold mb-4">
          Ready to get started?
        </h2>
        <p class="text-lg mb-8">
          Join thousands of users today
        </p>
        <UButton size="lg" color="primary">
          Sign Up Now
        </UButton>
      </div>
    </UContainer>

`)

	default:
		// Generic section
		sb.WriteString(`    <UContainer class="py-16">
      <h2 class="text-3xl font-bold mb-8 text-center">
        ` + section.Name + `
      </h2>
      <p class="text-center text-gray-600 dark:text-gray-400">
        ` + section.Content + `
      </p>
    </UContainer>

`)
	}

	return sb.String()
}

// validateBuildOutput validates the generated static site
func (is *ImplementationSpecialist) validateBuildOutput(generatedPath string, log *strings.Builder) ValidationResult {
	result := ValidationResult{
		IsValid:   true,
		HTMLValid: true,
		CSSValid:  true,
		JSValid:   true,
		Errors:    []string{},
		Warnings:  []string{},
	}

	// Check if .output/public exists
	if _, err := os.Stat(generatedPath); os.IsNotExist(err) {
		result.IsValid = false
		result.Errors = append(result.Errors, "Generated path does not exist: "+generatedPath)
		log.WriteString("❌ .output/public directory not found\n")
		return result
	}

	log.WriteString("✓ .output/public directory exists\n")

	// Check for index.html
	indexPath := filepath.Join(generatedPath, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		result.HTMLValid = false
		result.IsValid = false
		result.Errors = append(result.Errors, "index.html not found")
		log.WriteString("❌ index.html not found\n")
	} else {
		log.WriteString("✓ index.html exists\n")
	}

	// Check for _nuxt directory (assets)
	nuxtDir := filepath.Join(generatedPath, "_nuxt")
	if _, err := os.Stat(nuxtDir); os.IsNotExist(err) {
		result.CSSValid = false
		result.JSValid = false
		result.Warnings = append(result.Warnings, "_nuxt directory not found")
		log.WriteString("⚠ _nuxt directory not found\n")
	} else {
		log.WriteString("✓ _nuxt directory exists\n")
	}

	if result.IsValid {
		log.WriteString("\n✅ Build validation passed\n")
	} else {
		log.WriteString("\n❌ Build validation failed\n")
	}

	return result
}

// buildErrorOutput builds an error output with partial results
func (is *ImplementationSpecialist) buildErrorOutput(projectPath, buildLog string, err error) (*ImplementationOutput, error) {
	return &ImplementationOutput{
		ProjectPath: projectPath,
		BuildLog:    buildLog,
		ValidationResults: ValidationResult{
			IsValid: false,
			Errors:  []string{err.Error()},
		},
		StaticSiteReady: false,
		CreatedAt:       time.Now(),
	}, err
}

// ValidateOutput validates the implementation output
func (is *ImplementationSpecialist) ValidateOutput(output interface{}) error {
	if err := is.BaseSpecialist.ValidateOutput(output); err != nil {
		return err
	}

	implOutput, ok := output.(*ImplementationOutput)
	if !ok {
		return fmt.Errorf("invalid output type: expected ImplementationOutput")
	}

	if implOutput.ProjectPath == "" {
		return fmt.Errorf("project path is empty")
	}

	if !implOutput.StaticSiteReady {
		return fmt.Errorf("static site is not ready")
	}

	if !implOutput.ValidationResults.IsValid {
		return fmt.Errorf("validation failed: %v", implOutput.ValidationResults.Errors)
	}

	return nil
}

// GetExpectedOutputSchema returns the expected output schema
func (is *ImplementationSpecialist) GetExpectedOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_path":      map[string]interface{}{"type": "string"},
			"generated_path":    map[string]interface{}{"type": "string"},
			"static_site_ready": map[string]interface{}{"type": "boolean"},
		},
		"required": []string{
			"project_path",
			"generated_path",
			"static_site_ready",
		},
	}
}

// ToJSON converts the specialist to JSON
func (is *ImplementationSpecialist) ToJSON() []byte {
	data := map[string]interface{}{
		"type":    is.Type,
		"name":    is.Name,
		"timeout": is.Timeout.String(),
	}
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return jsonData
}
