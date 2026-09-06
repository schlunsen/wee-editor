package packs

// gstackSkillBody returns a delegating skill body that reads the actual gstack SKILL.md from disk.
// gstack is already installed at pack install time, so we just delegate directly.
func gstackSkillBody(skillName, description string) string {
	return `Read the full skill instructions and follow them exactly:

` + "```bash" + `
GSTACK_DIR=""
[ -d ".claude/skills/gstack/` + skillName + `" ] && GSTACK_DIR=".claude/skills/gstack"
[ -z "$GSTACK_DIR" ] && [ -d "$HOME/.claude/skills/gstack/` + skillName + `" ] && GSTACK_DIR="$HOME/.claude/skills/gstack"
if [ -n "$GSTACK_DIR" ]; then
  cat "$GSTACK_DIR/` + skillName + `/SKILL.md"
else
  echo "ERROR: gstack not found. Reinstall the GStack pack from the Library."
fi
` + "```" + `

Follow the SKILL.md instructions from the output above. ` + description + `
`
}

// gstackSetupCommand is the shell command run during pack installation.
// It clones the gstack repo into ~/.claude/skills/gstack as the global source.
// Per-project enablement copies SKILL.md files via EnablePackForProject.
const gstackSetupCommand = `set -e
GSTACK_DIR="$HOME/.claude/skills/gstack"
if [ -d "$GSTACK_DIR" ]; then
  echo "gstack already installed at $GSTACK_DIR, updating..."
  cd "$GSTACK_DIR" && git pull --ff-only 2>/dev/null || true
else
  mkdir -p "$HOME/.claude/skills"
  git clone https://github.com/garrytan/gstack.git "$GSTACK_DIR"
  cd "$GSTACK_DIR" && ./setup
fi
echo "gstack installed successfully at $GSTACK_DIR"
`

// GstackSkillDirs returns the list of gstack skill directory names.
// Used by EnablePackForProject to know which SKILL.md files to copy.
func GstackSkillDirs() []string {
	return []string{
		"autoplan", "benchmark", "browse", "canary", "careful", "codex",
		"connect-chrome", "cso", "design-consultation", "design-review",
		"document-release", "freeze", "gstack-upgrade", "guard", "investigate",
		"land-and-deploy", "office-hours", "plan-ceo-review", "plan-design-review",
		"plan-eng-review", "qa", "qa-only", "retro", "review",
		"setup-browser-cookies", "setup-deploy", "ship", "unfreeze",
	}
}

func gstackPack() *Pack {
	return &Pack{
		Name:         "gstack",
		DisplayName:  "GStack by Garry Tan",
		Description:  "28 specialized AI skills that turn Claude Code into a virtual engineering team. Covers the full sprint cycle: planning, building, reviewing, QA testing with real browsers, shipping, and retrospectives. Created by Garry Tan.",
		Version:         "1.0.0",
		Category:        "workflow",
		Icon:            "factory",
		SetupCommand:    gstackSetupCommand,
		RequiresProject: false,
		Skills: []PackSkill{
			// --- Planning ---
			{
				Name:        "office-hours",
				Description: "YC Office Hours: six forcing questions that expose the real problem before you write code",
				Body: gstackSkillBody("office-hours",
					"YC Office Hours: two modes. Startup mode asks six forcing questions that expose the real problem. Engineering mode stress-tests your approach before you build."),
				Effort: "medium",
				Agent:  "Plan",
			},
			{
				Name:        "plan-ceo-review",
				Description: "CEO/founder-mode plan review: rethink the problem, find the 10-star product",
				Body: gstackSkillBody("plan-ceo-review",
					"CEO/founder-mode plan review. Rethink the problem, find the 10-star product, challenge scope and ambition."),
				Effort: "medium",
				Agent:  "Plan",
			},
			{
				Name:        "plan-eng-review",
				Description: "Eng manager-mode plan review: lock in architecture, tests, and execution plan",
				Body: gstackSkillBody("plan-eng-review",
					"Eng manager-mode plan review. Lock in the execution plan: architecture, test strategy, edge cases, and risk mitigation."),
				Effort: "medium",
				Agent:  "Plan",
			},
			{
				Name:        "plan-design-review",
				Description: "Designer's eye plan review: interactive design critique",
				Body: gstackSkillBody("plan-design-review",
					"Designer's eye plan review: interactive critique for UI/UX gaps, accessibility, and visual consistency."),
				Effort: "medium",
				Agent:  "Plan",
			},
			{
				Name:        "autoplan",
				Description: "Auto-review pipeline: runs CEO, design, and eng reviews in sequence",
				Body: gstackSkillBody("autoplan",
					"Auto-review pipeline that reads and runs the full CEO, design, and eng review skills from disk in sequence."),
				Effort: "max",
				Agent:  "Plan",
			},

			// --- Development ---
			{
				Name:        "design-consultation",
				Description: "Design consultation: research the landscape and propose a design system",
				Body: gstackSkillBody("design-consultation",
					"Design consultation: understands your product, researches the landscape, proposes a design system with color palette, typography, and component inventory."),
				Effort: "high",
			},
			{
				Name:        "investigate",
				Description: "Systematic debugging: four phases from investigation to root cause fix",
				Body: gstackSkillBody("investigate",
					"Systematic debugging with root cause investigation. Four phases: investigate, analyze, hypothesize, implement. Iron Law: no fixes without root cause."),
				Effort: "high",
			},
			{
				Name:        "codex",
				Description: "OpenAI Codex CLI wrapper for independent code review",
				Body: gstackSkillBody("codex",
					"OpenAI Codex CLI wrapper with three modes: code review (independent diff review), second opinion (cross-validate Claude's work), and direct task execution."),
				Effort: "medium",
			},

			// --- Review & QA ---
			{
				Name:        "review",
				Description: "Pre-landing PR review: SQL safety, LLM trust boundaries, performance",
				Body: gstackSkillBody("review",
					"Pre-landing PR review. Analyzes diff against the base branch for SQL safety, LLM trust boundaries, performance, and correctness."),
				Effort: "high",
			},
			{
				Name:        "design-review",
				Description: "Designer's eye QA: finds visual inconsistency, spacing issues, hierarchy problems",
				Body: gstackSkillBody("design-review",
					"Designer's eye QA: finds visual inconsistency, spacing issues, hierarchy problems, and accessibility gaps using the headless browser."),
				Effort: "high",
			},
			{
				Name:        "qa",
				Description: "Systematically QA test a web app and fix bugs found (requires browser)",
				Body: gstackSkillBody("qa",
					"Systematically QA test a web application and fix bugs found. Runs QA testing with a real headless Chromium browser, then auto-fixes issues."),
				Effort: "max",
			},
			{
				Name:        "qa-only",
				Description: "Report-only QA testing: produces findings without auto-fixing",
				Body: gstackSkillBody("qa-only",
					"Report-only QA testing. Systematically tests a web application and produces a findings report without making any code changes."),
				Effort: "high",
			},
			{
				Name:        "browse",
				Description: "Fast headless browser for QA testing and site dogfooding",
				Body: gstackSkillBody("browse",
					"Fast headless browser (~100ms per command). Navigate URLs, interact with elements, take screenshots, test forms, check responsive layouts, and assert states."),
				Effort: "medium",
			},
			{
				Name:        "benchmark",
				Description: "Performance regression detection using the headless browser",
				Body: gstackSkillBody("benchmark",
					"Performance regression detection using the browse daemon. Establishes baselines and detects regressions in page load times and interactions."),
				Effort: "high",
			},

			// --- Release ---
			{
				Name:        "ship",
				Description: "Ship workflow: merge base, test, review diff, bump version, create PR",
				Body: gstackSkillBody("ship",
					"Ship workflow: detect + merge base branch, run tests, review diff, bump VERSION, update CHANGELOG, commit, push, create PR."),
				Effort: "high",
			},
			{
				Name:        "land-and-deploy",
				Description: "Land and deploy: merge PR, wait for CI, verify deployment",
				Body: gstackSkillBody("land-and-deploy",
					"Land and deploy workflow. Merges the PR, waits for CI and deploy, then verifies the deployment is healthy."),
				Effort: "high",
			},
			{
				Name:        "canary",
				Description: "Post-deploy canary monitoring for console errors and regressions",
				Body: gstackSkillBody("canary",
					"Post-deploy canary monitoring. Watches the live app for console errors, failed network requests, and visual regressions."),
				Effort: "medium",
			},
			{
				Name:        "setup-deploy",
				Description: "Configure deployment settings for land-and-deploy",
				Body: gstackSkillBody("setup-deploy",
					"Configure deployment settings for /land-and-deploy. Detects your deploy platform and writes the config."),
				Effort: "low",
			},
			{
				Name:        "document-release",
				Description: "Post-ship documentation update: cross-references changes with docs",
				Body: gstackSkillBody("document-release",
					"Post-ship documentation update. Reads all project docs, cross-references the latest changes, and updates documentation to match."),
				Effort: "medium",
			},

			// --- Safety ---
			{
				Name:        "careful",
				Description: "Safety guardrails: warns before destructive commands (rm -rf, DROP TABLE, force-push)",
				Body: gstackSkillBody("careful",
					"Safety guardrails for destructive commands. Warns before rm -rf, DROP TABLE, force-push, git reset --hard, kubectl delete, and similar."),
				Effort: "low",
			},
			{
				Name:        "freeze",
				Description: "Restrict file edits to a specific directory for the session",
				Body: gstackSkillBody("freeze",
					"Restrict file edits to a specific directory for the session. Blocks Edit and Write outside the allowed path."),
				Effort: "low",
			},
			{
				Name:        "unfreeze",
				Description: "Clear the freeze boundary, allowing edits to all directories again",
				Body: gstackSkillBody("unfreeze",
					"Clear the freeze boundary set by /freeze, allowing edits to all directories again."),
				Effort: "low",
			},
			{
				Name:        "guard",
				Description: "Full safety mode: destructive command warnings + directory-scoped edits",
				Body: gstackSkillBody("guard",
					"Full safety mode combining /careful (destructive command warnings) with /freeze (directory-scoped edits). Maximum safety for production work."),
				Effort: "low",
			},

			// --- Utilities ---
			{
				Name:        "retro",
				Description: "Weekly engineering retrospective with commit analysis and trend tracking",
				Body: gstackSkillBody("retro",
					"Weekly engineering retrospective. Analyzes commit history, work patterns, and code quality metrics with persistent history and trend tracking."),
				Effort: "medium",
			},
			{
				Name:        "cso",
				Description: "Chief Security Officer mode: OWASP Top 10 + STRIDE threat modeling",
				Body: gstackSkillBody("cso",
					"Chief Security Officer mode. Infrastructure-first security audit: secrets archaeology, OWASP Top 10, STRIDE threat modeling, and remediation."),
				Effort: "high",
			},
			{
				Name:        "connect-chrome",
				Description: "Launch real Chrome controlled by gstack with the Side Panel extension",
				Body: gstackSkillBody("connect-chrome",
					"Launch real Chrome controlled by gstack with the Side Panel extension auto-loaded for interactive browser testing."),
				Effort: "low",
			},
			{
				Name:        "setup-browser-cookies",
				Description: "Import cookies from your real browser into the headless session",
				Body: gstackSkillBody("setup-browser-cookies",
					"Import cookies from your real Chromium browser (Chrome, Brave, Arc, Edge) into the headless browse session."),
				Effort: "low",
			},
			{
				Name:        "gstack-upgrade",
				Description: "Upgrade gstack to the latest version",
				Body: gstackSkillBody("gstack-upgrade",
					"Upgrade gstack to the latest version. Detects global vs vendored install and handles the upgrade accordingly."),
				Effort: "low",
			},
		},
	}
}
