package packs

// BuiltinPacks returns all pre-built packs that ship with Wee
func BuiltinPacks() []*Pack {
	return []*Pack{
		safetyPack(),
		qualityPack(),
		devopsPack(),
		reviewPack(),
		siteGeneratorPack(),
		diagramPack(),
		gstackPack(),
		eccPack(),
	}
}

func safetyPack() *Pack {
	return &Pack{
		Name:        "safety",
		DisplayName: "Safety Pack",
		Description: "Protect against dangerous commands, secret leaks, and destructive operations. Essential guardrails for any AI-assisted development workflow.",
		Version:     "1.0.0",
		Category:    "safety",
		Icon:        "shield",
		Hooks: []PackHook{
			{
				EventName:   "PreToolUse",
				Matcher:     "Bash",
				Type:        "command",
				Command:     `bash -c 'INPUT=$(cat); CMD=$(printf "%s" "$INPUT" | jq -r ".tool_input.command // empty"); if printf "%s" "$CMD" | grep -qEi "rm\s+-rf\s+/[^.]|rm\s+-rf\s+~|git\s+push\s+--force\s+(origin\s+)?(main|master)|DROP\s+TABLE|DROP\s+DATABASE|truncate\s+table|:(){ :|:& };:|mkfs\.|dd\s+if=.+of=/dev/"; then echo "{\"decision\":\"block\",\"reason\":\"Blocked dangerous command: potential destructive operation detected\"}"; else echo "{\"decision\":\"allow\"}"; fi'`,
				Timeout:     5,
				Description: "Block dangerous shell commands (rm -rf /, force push to main, DROP TABLE, etc.)",
			},
			{
				EventName:   "PreToolUse",
				Matcher:     "Write|Edit",
				Type:        "command",
				Command:     `bash -c 'INPUT=$(cat); FILE=$(printf "%s" "$INPUT" | jq -r ".tool_input.file_path // .tool_input.path // empty"); if printf "%s" "$FILE" | grep -qEi "\\.env$|\\.env\\.|credentials|secret|private.key$|id_rsa$|id_ed25519$|\\.pem$|token|password"; then printf "{\"decision\":\"block\",\"reason\":\"Blocked: refusing to modify sensitive file\"}\n"; else echo "{\"decision\":\"allow\"}"; fi'`,
				Timeout:     5,
				Description: "Protect sensitive files (.env, credentials, private keys) from edits",
			},
			{
				EventName:   "PostToolUse",
				Matcher:     "Write|Edit",
				Type:        "command",
				Command:     `bash -c 'INPUT=$(cat); CONTENT=$(printf "%s" "$INPUT" | jq -r ".tool_result.content // empty"); if printf "%s" "$CONTENT" | grep -qEi "(sk|pk|api|token|key|secret|password|bearer)[-_]?[A-Za-z0-9]{20,}|AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{36}|xox[bpas]-[A-Za-z0-9-]+"; then echo "WARNING: Possible secret or API key detected in file content. Please review and remove before committing." >&2; fi'`,
				Timeout:     5,
				Description: "Scan written files for accidentally leaked secrets and API keys",
			},
			{
				EventName:   "PreToolUse",
				Matcher:     "Bash",
				Type:        "command",
				Command:     `bash -c 'INPUT=$(cat); CMD=$(printf "%s" "$INPUT" | jq -r ".tool_input.command // empty"); if printf "%s" "$CMD" | grep -qEi "git\s+(reset\s+--hard|clean\s+-fd|checkout\s+--\s+\\.|branch\s+-D)"; then echo "{\"decision\":\"block\",\"reason\":\"Blocked: destructive git operation. Use safer alternatives (git stash, git branch -d, etc.)\"}"; else echo "{\"decision\":\"allow\"}"; fi'`,
				Timeout:     5,
				Description: "Warn on destructive git operations (reset --hard, clean -fd, checkout --, branch -D)",
			},
		},
	}
}

func qualityPack() *Pack {
	return &Pack{
		Name:        "quality",
		DisplayName: "Quality Pack",
		Description: "Automated code quality checks with linting, formatting, and the simplify skill for cleaner code.",
		Version:     "1.0.0",
		Category:    "quality",
		Icon:        "sparkles",
		Skills: []PackSkill{
			{
				Name:        "lint",
				Description: "Run the project linter and fix issues automatically",
				Body: `When the user invokes this skill, perform the following steps:

1. **Detect the project type** by checking for configuration files:
   - JavaScript/TypeScript: Look for package.json, .eslintrc*, tsconfig.json
   - Python: Look for pyproject.toml, setup.py, .flake8, ruff.toml
   - Go: Look for go.mod
   - Rust: Look for Cargo.toml

2. **Run the appropriate linter**:
   - JS/TS: Run ` + "`npx eslint --fix .`" + ` or ` + "`npm run lint -- --fix`" + `
   - Python: Run ` + "`ruff check --fix .`" + ` or ` + "`flake8 .`" + `
   - Go: Run ` + "`golangci-lint run --fix`" + ` or ` + "`go vet ./...`" + `
   - Rust: Run ` + "`cargo clippy --fix --allow-dirty`" + `

3. **Report results**: Show a summary of issues found and fixed.
   If there are remaining issues that couldn't be auto-fixed, list them.

4. **If no linter is configured**, suggest setting one up for the project.`,
				ArgumentHint: "[path or pattern]",
				Effort:       "low",
			},
			{
				Name:        "simplify",
				Description: "Review changed code for reuse, quality, and efficiency, then fix issues",
				Body: `Review all recently changed code in the current branch for:

1. **Code reuse**: Identify duplicated logic that could be extracted into shared functions
2. **Unnecessary complexity**: Simplify overly complex conditionals, loops, or abstractions
3. **Dead code**: Find unused variables, functions, imports, or unreachable branches
4. **Naming**: Suggest clearer names for variables, functions, and types
5. **Performance**: Spot obvious inefficiencies (N+1 queries, unnecessary allocations)

For each issue found:
- Explain what the issue is
- Show the fix
- Apply the fix directly

Focus on the diff between the current branch and main/master. Only review changed files.`,
				Effort: "medium",
			},
		},
		Hooks: []PackHook{
			{
				EventName:   "PostToolUse",
				Matcher:     "Write|Edit",
				Type:        "command",
				Command:     `bash -c 'INPUT=$(cat); FILE=$(printf "%s" "$INPUT" | jq -r ".tool_input.file_path // .tool_input.path // empty"); case "$FILE" in */*) ;; *) exit 0;; esac; EXT="${FILE##*.}"; case "$EXT" in js|jsx|ts|tsx|json|css|scss|html|md|yaml|yml) npx prettier --write "$FILE" 2>/dev/null;; py) python -m black "$FILE" 2>/dev/null || python -m autopep8 --in-place "$FILE" 2>/dev/null;; go) gofmt -w "$FILE" 2>/dev/null;; rs) rustfmt "$FILE" 2>/dev/null;; esac'`,
				Timeout:     15,
				Description: "Auto-format files after edits using the appropriate formatter (Prettier, Black, gofmt, rustfmt)",
			},
			{
				EventName:   "Stop",
				Matcher:     "",
				Type:        "command",
				Command:     `bash -c 'if [ -f go.mod ]; then go vet ./... 2>&1 | head -20; elif [ -f package.json ]; then npx eslint . --max-warnings=0 2>&1 | tail -5; elif [ -f pyproject.toml ] || [ -f setup.py ]; then ruff check . 2>&1 | tail -5; fi'`,
				Timeout:     30,
				Description: "Run lint check before session completes to verify no errors in changed files",
			},
		},
	}
}

func devopsPack() *Pack {
	return &Pack{
		Name:        "devops",
		DisplayName: "DevOps Pack",
		Description: "Deployment, rollback, and service monitoring skills with safety hooks for production operations.",
		Version:     "1.0.0",
		Category:    "devops",
		Icon:        "rocket",
		Skills: []PackSkill{
			{
				Name:        "deploy",
				Description: "Deploy the application to a specified environment with safety checks",
				Body: `Deploy the application following these steps:

1. **Pre-flight checks**:
   - Verify the current branch is clean (no uncommitted changes)
   - Run tests: ` + "`make test`" + ` or ` + "`npm test`" + ` or the project's test command
   - Check that the build succeeds: ` + "`make build`" + ` or ` + "`npm run build`" + `

2. **Determine the environment** from the argument ($ARGUMENTS):
   - If "production" or "prod": Extra confirmation required, deploy to production
   - If "staging": Deploy to staging environment
   - If empty or "dev": Deploy to development environment

3. **Execute deployment**:
   - Look for deployment scripts: Makefile targets, package.json scripts, deploy.sh
   - Common patterns: ` + "`make deploy ENV=$ARGUMENTS`" + `, ` + "`npm run deploy:$ARGUMENTS`" + `
   - If using Docker: ` + "`docker compose up -d`" + `
   - If using Kubernetes: ` + "`kubectl apply -f k8s/`" + `

4. **Post-deploy verification**:
   - Check health endpoint if available
   - Show deployment status and URL

IMPORTANT: For production deployments, always show what will be deployed and ask for confirmation before proceeding.`,
				ArgumentHint: "[environment: dev|staging|prod]",
				Effort:       "high",
			},
			{
				Name:        "rollback",
				Description: "Rollback to a previous deployment version",
				Body: `Rollback the deployment to a previous version:

1. **Identify the current version** and the target version from $ARGUMENTS
2. **Check available versions/tags**: ` + "`git tag --sort=-v:refname | head -10`" + `
3. **Execute rollback**:
   - If Makefile: ` + "`make rollback VERSION=$ARGUMENTS`" + `
   - If Docker: ` + "`docker compose down && git checkout $ARGUMENTS && docker compose up -d`" + `
   - If Kubernetes: ` + "`kubectl rollout undo deployment/app`" + `
   - If git-based: ` + "`git revert HEAD`" + ` or ` + "`git checkout $ARGUMENTS`" + `
4. **Verify rollback**: Check that the service is healthy after rollback`,
				ArgumentHint: "[version or tag]",
				Effort:       "high",
			},
			{
				Name:        "status",
				Description: "Check service health and deployment status",
				Body: `Check the current deployment status:

1. **Service health**:
   - Check if the service is running (process, Docker container, k8s pod)
   - Hit the health endpoint if available
   - Check resource usage (CPU, memory)

2. **Deployment info**:
   - Current deployed version/commit
   - Last deployment time
   - Environment variables (sanitized, no secrets)

3. **Infrastructure**:
   - Docker: ` + "`docker compose ps`" + `
   - Kubernetes: ` + "`kubectl get pods`" + `
   - Systemd: ` + "`systemctl status`" + `

4. **Recent logs** (last 20 lines):
   - Docker: ` + "`docker compose logs --tail=20`" + `
   - Kubernetes: ` + "`kubectl logs --tail=20`" + `
   - Systemd: ` + "`journalctl -u service --no-pager -n 20`" + `

Display a summary with status indicators (healthy/degraded/down).`,
				Effort: "low",
			},
		},
		Hooks: []PackHook{
			{
				EventName:   "PreToolUse",
				Matcher:     "Bash",
				Type:        "command",
				Command:     `bash -c 'INPUT=$(cat); CMD=$(printf "%s" "$INPUT" | jq -r ".tool_input.command // empty"); if printf "%s" "$CMD" | grep -qEi "kubectl\s+delete|helm\s+uninstall|terraform\s+destroy|docker\s+system\s+prune"; then echo "{\"decision\":\"block\",\"reason\":\"Blocked: production-level destructive command requires manual execution\"}"; else echo "{\"decision\":\"allow\"}"; fi'`,
				Timeout:     5,
				Description: "Block production-destructive commands (kubectl delete, terraform destroy, etc.)",
			},
		},
	}
}

func reviewPack() *Pack {
	return &Pack{
		Name:        "review",
		DisplayName: "Review Pack",
		Description: "Comprehensive pull request review and test coverage analysis skills for better code quality.",
		Version:     "1.0.0",
		Category:    "review",
		Icon:        "clipboard",
		Skills: []PackSkill{
			{
				Name:        "review-pr",
				Description: "Perform a comprehensive pull request review",
				Body: `Perform a thorough code review of the specified pull request:

1. **Get PR context**: Use ` + "`gh pr view $ARGUMENTS`" + ` and ` + "`gh pr diff $ARGUMENTS`" + `

2. **Review categories** (check each):
   - **Correctness**: Logic errors, edge cases, off-by-one errors
   - **Security**: SQL injection, XSS, CSRF, hardcoded secrets, auth bypasses
   - **Performance**: N+1 queries, unnecessary allocations, missing indices
   - **Code style**: Naming conventions, code organization, DRY violations
   - **Testing**: Are new features tested? Are edge cases covered?
   - **Documentation**: Are public APIs documented? Are complex algorithms explained?
   - **Error handling**: Are errors properly caught, logged, and surfaced?

3. **For each issue found**, provide:
   - Severity: critical / warning / suggestion / nit
   - File and line number
   - What the issue is
   - Suggested fix (with code if applicable)

4. **Summary**: Overall assessment (approve / request changes / comment)
   - List of critical issues that must be fixed
   - List of suggestions for improvement
   - Positive callouts for good patterns

Use ` + "`gh pr review $ARGUMENTS`" + ` to submit the review if requested.`,
				ArgumentHint: "[PR number]",
				Context:      "fork",
				Effort:       "high",
			},
			{
				Name:        "test-coverage",
				Description: "Analyze test coverage and generate tests for uncovered code",
				Body: `Analyze test coverage and generate missing tests:

1. **Run coverage analysis**:
   - Go: ` + "`go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`" + `
   - JS/TS: ` + "`npx jest --coverage`" + ` or ` + "`npx vitest --coverage`" + `
   - Python: ` + "`pytest --cov --cov-report=term-missing`" + `

2. **Identify uncovered areas** (if $ARGUMENTS is specified, focus on that file/package):
   - Functions with 0% coverage
   - Complex branches not covered
   - Error handling paths not tested

3. **Generate tests** for the most critical uncovered code:
   - Unit tests for pure functions
   - Integration tests for API endpoints
   - Edge case tests for complex logic
   - Error path tests

4. **Verify** the new tests pass and improve coverage.

Focus on meaningful tests, not just coverage numbers. Prioritize:
- Business logic over boilerplate
- Error paths over happy paths (they're usually less tested)
- Public APIs over internal helpers`,
				ArgumentHint: "[file or package path]",
				Effort:       "high",
			},
		},
		Hooks: []PackHook{
			{
				EventName:   "PostToolUse",
				Matcher:     "Edit|Write",
				Type:        "command",
				Command:     `bash -c 'INPUT=$(cat); FILE=$(printf "%s" "$INPUT" | jq -r ".tool_input.file_path // .tool_input.path // empty"); case "$FILE" in */*) ;; *) exit 0;; esac; DIR=$(dirname "$FILE"); if [ -f go.mod ]; then PKG="./$(dirname "$FILE")/..."; go test "$PKG" -count=1 -timeout=30s 2>&1 | tail -5; elif [ -f package.json ]; then npx jest --findRelatedTests "$FILE" --passWithNoTests 2>&1 | tail -5; elif [ -f pyproject.toml ]; then pytest "$DIR" -x -q 2>&1 | tail -5; fi'`,
				Timeout:     60,
				Description: "Auto-run relevant tests after code changes",
			},
			{
				EventName:   "Stop",
				Matcher:     "",
				Type:        "prompt",
				Prompt:      "Summarize all changes made in this session. List files modified, features added/changed, and any remaining TODOs.",
				Description: "Generate a summary of all changes made in the session",
			},
		},
	}
}

func siteGeneratorPack() *Pack {
	return &Pack{
		Name:        "site-generator",
		DisplayName: "Site Generator Pack",
		Description: "Multi-agent site generation with specialist skills for design, implementation, and full orchestration.",
		Version:     "1.0.0",
		Category:    "site-generator",
		Icon:        "globe",
		Skills: []PackSkill{
			{
				Name:        "design",
				Description: "Design a website layout and visual style based on a brief",
				Body: `You are a web design specialist. Given a design brief, create a detailed design specification:

1. **Analyze the brief** ($ARGUMENTS): Understand the purpose, audience, and style preferences
2. **Create a design system**:
   - Color palette (primary, secondary, accent, neutrals) with hex values
   - Typography choices (heading font, body font, sizes, line heights)
   - Spacing scale (based on 4px or 8px grid)
   - Border radius, shadow, and elevation tokens

3. **Layout specification**:
   - Page structure (header, hero, sections, footer)
   - Grid system (12-column, content width, breakpoints)
   - Component inventory (buttons, cards, forms, navigation)
   - Responsive behavior for mobile/tablet/desktop

4. **Output**: A structured markdown document with:
   - CSS custom properties for the design tokens
   - Component HTML/CSS sketches
   - Layout wireframe descriptions
   - Interaction notes (hover states, transitions, animations)

Focus on modern, accessible design. Use semantic HTML and CSS Grid/Flexbox.`,
				ArgumentHint: "[design brief or description]",
				Agent:        "Plan",
				Effort:       "high",
			},
			{
				Name:        "implement",
				Description: "Implement a website design from a specification",
				Body: `You are a frontend implementation specialist. Given a design specification, build it:

1. **Read the design spec**: Look for design documents, mockups, or specifications
2. **Set up the project** (if not already set up):
   - Create index.html with semantic structure
   - Create styles.css with design tokens and base styles
   - Create script.js for interactions (if needed)

3. **Build components**:
   - Implement each component from the design spec
   - Use CSS Grid and Flexbox for layouts
   - Add responsive media queries
   - Implement hover states and transitions

4. **Quality checks**:
   - Validate HTML structure
   - Check accessibility (alt tags, ARIA labels, contrast)
   - Test responsive behavior
   - Optimize images and assets

5. **Output**: Working HTML/CSS/JS files ready for deployment.

Write clean, well-commented code. Use modern CSS features (custom properties, clamp(), container queries where appropriate).`,
				ArgumentHint: "[spec file or description]",
				Effort:       "high",
			},
			{
				Name:        "generate-site",
				Description: "Generate a complete website from a description using multi-agent orchestration",
				Body: `Orchestrate the full site generation process:

1. **Understand requirements** from $ARGUMENTS:
   - What type of site (landing page, portfolio, blog, docs, SaaS)
   - Key features and pages needed
   - Style preferences (modern, minimal, bold, corporate, playful)

2. **Phase 1 - Design**:
   - Create the design system and visual specification
   - Define the color palette, typography, and component library
   - Plan the page structure and layout

3. **Phase 2 - Implement**:
   - Build the HTML structure following the design spec
   - Implement CSS styles with the design tokens
   - Add JavaScript for interactions and dynamic features

4. **Phase 3 - Polish**:
   - Add responsive design for all breakpoints
   - Optimize for performance (minimize CSS, lazy load images)
   - Add meta tags for SEO
   - Add favicon and Open Graph tags
   - Ensure accessibility compliance

5. **Phase 4 - Deliver**:
   - Create a README with setup instructions
   - List any external dependencies
   - Provide deployment suggestions

The final output should be a complete, production-ready static site.`,
				ArgumentHint: "[site description]",
				Effort:       "max",
			},
		},
		Hooks: []PackHook{
			{
				EventName:   "PostToolUse",
				Matcher:     "Write",
				Type:        "command",
				Command:     `bash -c 'INPUT=$(cat); FILE=$(printf "%s" "$INPUT" | jq -r ".tool_input.file_path // .tool_input.path // empty"); case "$FILE" in */*) ;; *) exit 0;; esac; EXT="${FILE##*.}"; if [ "$EXT" = "html" ]; then if command -v tidy >/dev/null 2>&1; then tidy -q -e "$FILE" 2>&1 | head -5; fi; fi'`,
				Timeout:     10,
				Description: "Validate generated HTML files for correctness",
			},
		},
	}
}
