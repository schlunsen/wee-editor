package packs

// fireworksSkillBody returns a delegating skill body that reads the actual SKILL.md from disk.
// fireworks-tech-graph is cloned at pack install time, so we delegate to the on-disk instructions.
func fireworksSkillBody() string {
	return `Read the full skill instructions from the fireworks-tech-graph SKILL.md and follow them exactly:

` + "```bash" + `
FTG_DIR=""
[ -d ".claude/skills/fireworks-tech-graph" ] && FTG_DIR=".claude/skills/fireworks-tech-graph"
[ -z "$FTG_DIR" ] && [ -d "$HOME/.claude/skills/fireworks-tech-graph" ] && FTG_DIR="$HOME/.claude/skills/fireworks-tech-graph"
if [ -n "$FTG_DIR" ]; then
  cat "$FTG_DIR/SKILL.md"
else
  echo "ERROR: fireworks-tech-graph not found. Reinstall the Diagram Pack from the Library."
fi
` + "```" + `

Follow the SKILL.md instructions from the output above to generate the requested diagram.

IMPORTANT: The skill has helper scripts in its scripts/ directory and style references in references/.
Use the same $FTG_DIR path to access them (e.g., $FTG_DIR/scripts/validate-svg.sh, $FTG_DIR/references/style-1-flat-icon.md).
`
}

// fireworksSetupCommand clones the fireworks-tech-graph repo into ~/.claude/skills/fireworks-tech-graph.
const fireworksSetupCommand = `set -e
FTG_DIR="$HOME/.claude/skills/fireworks-tech-graph"
if [ -d "$FTG_DIR" ]; then
  echo "fireworks-tech-graph already installed at $FTG_DIR, updating..."
  cd "$FTG_DIR" && git pull --ff-only 2>/dev/null || true
else
  mkdir -p "$HOME/.claude/skills"
  git clone https://github.com/yizhiyanhua-ai/fireworks-tech-graph.git "$FTG_DIR"
fi
# Verify rsvg-convert is available (required dependency)
if ! command -v rsvg-convert >/dev/null 2>&1; then
  echo ""
  echo "WARNING: rsvg-convert not found. Install it for PNG export:"
  echo "  macOS:  brew install librsvg"
  echo "  Ubuntu: sudo apt-get install librsvg2-bin"
  echo ""
fi
echo "fireworks-tech-graph installed successfully at $FTG_DIR"
`

func diagramPack() *Pack {
	return &Pack{
		Name:         "diagram",
		DisplayName:  "Diagram Pack",
		Description:  "Production-quality SVG + PNG technical diagram generation. Supports architecture, data flow, flowchart, sequence, UML, ER, and more with 7 visual styles. Powered by fireworks-tech-graph.",
		Version:      "1.0.0",
		Category:     "diagram",
		Icon:         "pencil",
		SetupCommand: fireworksSetupCommand,
		Skills: []PackSkill{
			{
				Name:         "diagram",
				Description:  "Generate technical diagrams as SVG+PNG (architecture, flowchart, sequence, UML, ER, data flow, mind map, timeline)",
				Body:         fireworksSkillBody(),
				ArgumentHint: "[diagram type + description]",
				Effort:       "high",
			},
		},
		Hooks: []PackHook{
			{
				EventName:   "PostToolUse",
				Matcher:     "Write",
				Type:        "command",
				Command:     `bash -c 'INPUT=$(cat); FILE=$(printf "%s" "$INPUT" | jq -r ".tool_input.file_path // .tool_input.path // empty"); case "$FILE" in *.svg) rsvg-convert "$FILE" -o /dev/null 2>&1 && echo "SVG valid" || echo "WARNING: SVG validation failed for $FILE" ;; esac'`,
				Timeout:     10,
				Description: "Auto-validate SVG files after generation using rsvg-convert",
			},
		},
	}
}
