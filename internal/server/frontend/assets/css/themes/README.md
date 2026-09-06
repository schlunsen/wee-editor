# Frontend Theme System

This directory contains all theme definitions for the Wee analytics dashboard. The theme system uses CSS custom properties (variables) for a clean, maintainable approach.

## Structure

```
themes/
├── README.md              # This file
├── _variables.css         # Default and light themes
├── neon.css               # Neon dark/light themes (Cyberpunk-inspired)
├── nord.css               # Nord dark/light themes (Arctic-inspired)
├── dracula.css            # Dracula dark/light themes (Vibrant purple)
├── southpark.css          # South Park dark/light themes (Comic-style)
└── wee.css                # Wee dark/light themes (Professional monochromatic)
```

## Adding a New Theme

To add a new theme:

1. **Create a new CSS file** in this directory (e.g., `mytheme.css`)

2. **Define CSS custom properties** for both dark and light variants:

```css
/* My Theme Dark */
[data-theme="mytheme-dark"] {
  --bg-primary: #1a1a1a;
  --bg-secondary: #2d2d2d;
  --bg-tertiary: #3a3a3a;
  --text-primary: #e8e8e8;
  --text-secondary: #b8b8b8;
  --text-muted: #888888;
  --accent-purple: #a78bfa;    /* Primary accent (used for buttons, links) */
  --accent-cyan: #67e8f9;      /* Secondary accent */
  --accent-green: #4ade80;     /* Success state */
  --accent-yellow: #fbbf24;    /* Warning state */
  --accent-orange: #ff9800;    /* Alternative accent */
  --border-color: #3a3a3a;
  --code-bg: #242424;
  --card-bg: #3a3a3a;
  --card-hover: #444444;
  --status-success: #4ade80;
  --status-warning: #fbbf24;
  --status-error: #f87171;

  /* Context Progress Bar Colors */
  --progress-start: #67e8f9;      /* 0-25% */
  --progress-low: #a78bfa;        /* 25-50% */
  --progress-medium: #fbbf24;     /* 50-75% */
  --progress-high: #f87171;       /* 75-100% */

  /* Typography */
  --font-primary: 'FontName', 'Fallback', sans-serif;
  --font-mono: 'MonoFont', monospace;
}

/* My Theme Light */
[data-theme="mytheme-light"] {
  /* Similar structure for light variant */
}
```

3. **Import the file** in `main.css`:

```css
@import './themes/mytheme.css';
```

4. **Register the theme** in `composables/useTheme.ts`:

```typescript
export type ThemeVariant = '...' | 'mytheme-dark' | 'mytheme-light'

export const availableThemes: Theme[] = [
  // ... existing themes
  {
    id: 'mytheme-dark',
    name: 'My Theme Dark',
    description: 'Description of your theme',
    isDark: true,
    fontFamily: 'FontName',
    fontDescription: 'Font description'
  },
  {
    id: 'mytheme-light',
    name: 'My Theme Light',
    description: 'Light variant description',
    isDark: false,
    fontFamily: 'FontName',
    fontDescription: 'Font description'
  }
]
```

5. **Add dark mode toggle support** in `composables/useTheme.ts`:

```typescript
const toggleDarkMode = () => {
  // ... existing code
  else if (currentId.includes('mytheme')) {
    currentTheme.value = isDark.value ? 'mytheme-light' : 'mytheme-dark'
  }
  // ... rest of code
}
```

6. **Add Google Font** (if needed) in `nuxt.config.ts`:

```typescript
// In the link array within app.head.meta
{ rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=YourFont:wght@400;500;600;700&display=swap' }
```

## CSS Variable Reference

### Background Colors
- `--bg-primary`: Main background color
- `--bg-secondary`: Secondary background (slightly lighter/darker)
- `--bg-tertiary`: Tertiary background (cards, containers)
- `--card-bg`: Card background color
- `--card-hover`: Card hover state background
- `--code-bg`: Code block background

### Text Colors
- `--text-primary`: Main text color
- `--text-secondary`: Secondary text (less emphasis)
- `--text-muted`: Muted/disabled text

### Accent Colors
- `--accent-purple`: Primary accent (buttons, links, highlights)
- `--accent-cyan`: Secondary accent
- `--accent-green`: Success/positive state
- `--accent-yellow`: Warning state
- `--accent-orange`: Alternative accent

### Status Colors
- `--status-success`: Success indicator
- `--status-warning`: Warning indicator
- `--status-error`: Error indicator

### Progress Bar Colors
Used for context usage bars and similar indicators:
- `--progress-start`: 0-25% usage
- `--progress-low`: 25-50% usage
- `--progress-medium`: 50-75% usage
- `--progress-high`: 75-100% usage

### Border & Structure
- `--border-color`: Border and divider color

### Typography
- `--font-primary`: Main font family
- `--font-mono`: Monospace font for code

## Current Themes

### Default (Inter font)
- **Dark**: Classic dark theme with purple accents
- **Light**: Clean light theme

### Neon (Orbitron font)
- **Dark**: Cyberpunk-inspired with bright neon colors
- **Light**: Bright neon accents on light background

### Nord (Fira Code/Fira Sans)
- **Dark**: Arctic-inspired cool blues and muted tones
- **Light**: Bright Nordic palette with subtle blues

### Dracula (JetBrains Mono)
- **Dark**: Vibrant purple and pink on dark background
- **Light**: Soft pastels with Dracula accent colors

### South Park (Comic Sans MS)
- **Dark**: Bold colors and comic book style
- **Light**: Comic book palette on light background

### Wee (Questrial)
- **Dark**: Professional dark theme with monochromatic gray palette
- **Light**: Clean light theme with monochromatic gray palette
- **Colors**: Professional palette using #DBDBDB, #4A4A4B, #848383

## Usage in Components

Use CSS variables in your Vue components:

```vue
<style scoped>
.my-component {
  background: var(--bg-primary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.button {
  background: var(--accent-purple);
  color: white;
}

.status-success {
  color: var(--status-success);
}
</style>
```

## Tips

1. **Always use CSS variables** instead of hardcoded colors for theme consistency
2. **Test both dark and light variants** when adding new theme elements
3. **Use semantic naming** (e.g., `--status-error` instead of `--red-color`)
4. **Document your theme** in this README when adding a new one
5. **Consider accessibility** - ensure sufficient color contrast ratios
6. **Use the progress bar colors** for context/resource usage indicators
