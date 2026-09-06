# Landing Page Generator Components

Phase 5 Frontend UI implementation for the Landing Page Generator multi-agent orchestration system.

## Overview

This directory contains Vue 3 components for the Landing Page Generator feature in the Analytics Dashboard. The system generates fully functional landing pages from natural language descriptions using a multi-agent orchestration workflow.

## Components

### 1. **LandingPageForm.vue**
Main form component for accepting user input to generate a landing page.

**Features:**
- Textarea for detailed landing page descriptions
- Category selector (SaaS, Portfolio, E-commerce, etc.)
- Style preferences (Modern, Minimal, Vibrant, Professional, Playful)
- Additional options (Dark mode, Mobile-first, Animations, SEO optimization)
- Real-time character counting with validation
- Example descriptions for quick-start
- Form validation with error messages

**Props:**
- `loading?: boolean` - Disable form while generating

**Emits:**
- `submit(formData)` - Fired when form is submitted with validation passed

### 2. **ProgressTimeline.vue**
Visual timeline showing the execution progress of the multi-agent workflow.

**Features:**
- Vertical timeline of execution steps
- Step status indicators (pending, running, completed, failed)
- Per-step elapsed time and estimated duration
- Specialist agent name badges
- Error messages and retry buttons for failed steps
- Overall progress bar with percentage
- Completion/failure summary

**Props:**
- `steps: Step[]` - Array of execution steps
- `currentStep?: number` - Current step number
- `status?: 'idle' | 'running' | 'completed' | 'failed'`
- `allowRetry?: boolean` - Show retry button on failures

**Emits:**
- `retry(stepNumber)` - User clicked retry for a step

**Step Interface:**
```typescript
interface Step {
  step_number: number
  name: string
  specialist_name?: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  elapsed_time?: number
  estimated_time?: number
  description?: string
  error?: string
  session_id?: string
}
```

### 3. **SpecialistPanel.vue**
Live panel showing current specialist agent's output and progress.

**Features:**
- Specialist name and role display
- Session ID with link to view in Live Agents
- Real-time agent output messages
- Toggle for input/output data inspection
- Debug mode for raw data viewing
- Auto-scrolling to latest messages

**Props:**
- `specialist` - Current specialist info
- `sessionId` - Agent session ID for this specialist

### 4. **ArtifactBrowser.vue**
File browser for viewing generated artifacts organized by type.

**Features:**
- Tabbed interface for artifact types (Design Spec, Template Files, Implementation, Optimization)
- Search and filter artifacts
- Sort by name, size, or date
- File icons and metadata display
- Bulk download as ZIP
- Preview button for each artifact

**Props:**
- `artifacts?: Artifact[]` - List of generated artifacts
- `projectId?: string` - Current project ID

**Emits:**
- `download(artifact)` - Download single artifact
- `preview(artifact)` - Preview artifact

### 5. **ArtifactPreview.vue**
Modal component for previewing different types of artifacts.

**Features:**
- HTML preview in iframe with device size selector
- Image preview
- Code/JSON preview with syntax highlighting
- Dark mode toggle for HTML previews
- Copy-to-clipboard for code
- Fullscreen mode
- Download button

**Props:**
- `artifact` - Artifact to preview

**Emits:**
- `close()` - Close preview modal
- `download(artifact)` - Download artifact

### 6. **GenerationMetrics.vue**
Statistics and metrics display for the generation process.

**Features:**
- Total elapsed time
- Tokens used with per-step breakdown
- Estimated cost with per-specialist breakdown
- Model and provider information
- Cost breakdown chart by specialist
- Detailed statistics table
- Comparison to average generation metrics

**Props:**
- `totalTokens?: number`
- `elapsedTime?: number`
- `estimatedTotalTime?: number`
- `totalSteps?: number`
- `completedSteps?: number`
- `failedSteps?: number`
- `specialists?: Specialist[]`
- `modelName?: string`
- `provider?: string`
- `showComparison?: boolean`

### 7. **ProjectCard.vue**
Card component for displaying landing page generation projects.

**Features:**
- Project title and status badge
- Project metadata (category, style, creation date)
- Progress bar for in-progress projects
- Generation metrics (time, cost, tokens)
- Preview of first 3 artifacts
- Action buttons (view, retry, download)

**Props:**
- `project` - Project data

**Emits:**
- `view()` - View project details
- `retry()` - Retry generation
- `download()` - Download project

## Composables

### useLandingPageGenerator.ts

Main composable for managing landing page generation state and WebSocket communication.

**Features:**
- WebSocket connection management with auto-reconnect
- Message handling for generation events
- Project loading and management
- Generation workflow control
- State persistence with localStorage
- Error handling and recovery

**Exported Functions:**
- `startGeneration(formData)` - Start new generation
- `getProject(projectId)` - Fetch project details
- `loadProjects()` - Load all projects
- `retryProject(projectId, stepNumber?)` - Retry generation
- `downloadProject(projectId)` - Download result
- `persistState()` - Save state to localStorage
- `restoreState()` - Load state from localStorage

**State Properties:**
- `currentGeneration` - Current generation project
- `projects` - List of all projects
- `isGenerating` - Whether generation is in progress
- `isConnected` - WebSocket connection status

## Type Definitions

Located in the composable:

```typescript
interface LandingPageProject {
  id: string
  user_description: string
  category: string
  style: string
  status: 'idle' | 'in_progress' | 'completed' | 'failed'
  created_at: string
  updated_at: string
  current_step?: number
  total_steps?: number
  steps?: LandingPageStep[]
  metrics?: GenerationMetrics
  artifacts?: Artifact[]
}

interface LandingPageStep {
  step_number: number
  name: string
  specialist_name: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  session_id?: string
  elapsed_time?: number
  estimated_time?: number
  description?: string
  error?: string
  output?: Record<string, any>
  input?: Record<string, any>
}

interface GenerationMetrics {
  total_time_seconds: number
  total_tokens: number
  estimated_cost: number
  completed_steps: number
  failed_steps: number
  current_step: number
}

interface Artifact {
  id: string
  project_id: string
  step_number?: number
  name: string
  type: string
  size: number
  url?: string
  content?: string
  created_at: string
}
```

## WebSocket Events

The composable handles these WebSocket event types:

- `landing_page_started` - Generation started
- `landing_page_step_started` - Step began execution
- `landing_page_step_completed` - Step finished successfully
- `landing_page_step_failed` - Step failed with error
- `landing_page_completed` - Generation finished successfully
- `landing_page_failed` - Generation failed
- `landing_page_artifact` - New artifact generated

## API Endpoints

Components interact with these endpoints via the composable:

- `POST /api/landing-page/generate` - Start generation
- `GET /api/landing-page/projects` - List projects
- `GET /api/landing-page/projects/:id` - Get project details
- `GET /api/landing-page/projects/:id/download` - Download result
- `POST /api/landing-page/projects/:id/retry` - Retry generation
- `GET /api/landing-page/artifacts/:id` - Get artifact content

## Styling

All components use Tailwind CSS with dark mode support. Shared utilities:

- Color scheme matches existing analytics dashboard
- Dark mode automatically enabled based on `dark:` prefixes
- Responsive design with mobile-first approach
- Consistent spacing and typography

## Responsive Design

Components support:
- **Desktop**: 1920px+ - Full layout with all sidebars
- **Tablet**: 768px-1919px - Adjusted panel widths
- **Mobile**: <768px - Stacked layout

## Accessibility

Components include:
- Semantic HTML tags
- ARIA labels for interactive elements
- Keyboard navigation support
- Color contrast compliance (WCAG 2.1 AA)
- Focus indicators for keyboard users
- Form validation with error messages

## Usage Examples

### Basic Generation Flow

```vue
<template>
  <LandingPageGenerator />
</template>

<script setup>
// Landing page generator is self-contained in the page component
</script>
```

### Manual Integration

```vue
<template>
  <div class="generator">
    <LandingPageForm
      @submit="handleSubmit"
      :loading="isGenerating"
    />

    <div v-if="currentGeneration">
      <ProgressTimeline
        :steps="currentGeneration.steps"
        :current-step="currentGeneration.current_step"
        @retry="handleRetry"
      />

      <ArtifactBrowser
        :artifacts="currentGeneration.artifacts"
        @download="downloadArtifact"
      />
    </div>
  </div>
</template>

<script setup>
import { useLandingPageGenerator } from '~/composables/useLandingPageGenerator'

const { currentGeneration, startGeneration, retryProject } = useLandingPageGenerator()

const handleSubmit = async (formData) => {
  await startGeneration(formData)
}

const handleRetry = async (stepNumber) => {
  await retryProject(currentGeneration.value!.id, stepNumber)
}
</script>
```

## State Persistence

The composable automatically:
1. Saves state to localStorage every 5 seconds
2. Restores state on page reload
3. Handles WebSocket reconnection with exponential backoff
4. Preserves generation progress across browser sessions

## Error Handling

Components gracefully handle:
- Network disconnections with auto-reconnect
- Generation failures with retry option
- Invalid form inputs with validation feedback
- File upload/download errors with user notification
- WebSocket message parsing errors

## Performance Considerations

- Lazy-loading of components where possible
- Efficient re-rendering with Vue 3 Composition API
- Memoized computed properties
- Debounced search/filter operations
- Virtual scrolling for long artifact lists

## Testing

Components are designed to be testable:
- Separated business logic in composables
- Props-driven rendering
- Emitted events for parent interaction
- Minimal DOM dependencies

## Browser Support

Works on all modern browsers:
- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Mobile browsers (iOS Safari 14+, Chrome Mobile)

## Future Enhancements

- [ ] Parallel specialist execution view
- [ ] A/B testing interface for design variants
- [ ] Direct deployment to hosting providers
- [ ] AI-generated video background preview
- [ ] SEO score display
- [ ] Accessibility scoring and reports
- [ ] Generation history with filtering
- [ ] Bulk generation scheduling
