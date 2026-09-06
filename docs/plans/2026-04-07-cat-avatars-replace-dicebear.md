# Cat Avatars Replace DiceBear — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the DiceBear external avatar system with local cat avatar images as the default fallback and seeded themes.

**Architecture:** The 36 cat images (12 characters x 3 styles) become the base avatar system. The frontend `useCharacterAvatar` composable maps session IDs to local cat image paths instead of DiceBear API URLs. The database seeder creates 3 cat themes (Neon Cats as default, Dali Cats, Line Cats) instead of 4 DiceBear themes. All DiceBear-specific code is removed.

**Tech Stack:** Go (backend seeder, handlers), TypeScript/Vue (frontend composables, components), SQLite (avatar themes/avatars), sips (image resizing)

---

### Task 1: Resize Cat Images to 128x128

**Files:**
- Modify: all 36 files in `internal/server/frontend/public/avatars/cats/*.jpg`

**Step 1: Batch resize all cat avatars**

```bash
cd internal/server/frontend/public/avatars/cats
for f in *.jpg; do
  sips -z 128 128 "$f"
done
```

**Step 2: Verify sizes**

```bash
ls -la *.jpg | awk '{print $5, $9}' | head -5
sips -g pixelWidth -g pixelHeight cat-coder.jpg
```

Expected: all files ~5-12KB, dimensions 128x128

**Step 3: Commit**

```bash
git add internal/server/frontend/public/avatars/cats/
git commit -m "chore: resize cat avatars to 128x128 for optimal UI rendering"
```

---

### Task 2: Rewrite useCharacterAvatar.ts — Cats Replace DiceBear Fallbacks

**Files:**
- Modify: `internal/server/frontend/app/composables/useCharacterAvatar.ts`

**Step 1: Rewrite the composable**

Replace the entire file with local cat avatar paths and TPB-inspired names:

```typescript
/**
 * Composable for mapping session IDs to cat-based fallback avatars
 * Uses TPB-inspired cat names and local avatar images
 */

export interface CharacterInfo {
  name: string
  avatar: string
  color: string
}

// Cat characters with Trailer Park Boys-inspired names
// Maps to local images in /avatars/cats/
const fallbackAvatars: CharacterInfo[] = [
  { name: 'Bubbles de Catt', avatar: '/avatars/cats/cat-coder.jpg', color: '#85C1E2' },
  { name: 'Julian Purresso', avatar: '/avatars/cats/cat-coffee.jpg', color: '#F8B88B' },
  { name: 'Inspector Lahisker', avatar: '/avatars/cats/cat-detective.jpg', color: '#F9E79F' },
  { name: 'J-Roc Meowski', avatar: '/avatars/cats/cat-dj.jpg', color: '#BB8FCE' },
  { name: 'Cyrus Rootpaw', avatar: '/avatars/cats/cat-hacker.jpg', color: '#4ECDC4' },
  { name: 'Conky Catsworth', avatar: '/avatars/cats/cat-music.jpg', color: '#F1948A' },
  { name: 'Shadow Rickitty', avatar: '/avatars/cats/cat-ninja.jpg', color: '#45B7D1' },
  { name: 'Captain Shittpaw', avatar: '/avatars/cats/cat-pirate.jpg', color: '#FF6B6B' },
  { name: 'Ricky Lawnchpurr', avatar: '/avatars/cats/cat-rocket.jpg', color: '#FFA07A' },
  { name: 'Randy Catnaps', avatar: '/avatars/cats/cat-sleeping.jpg', color: '#D7BDE2' },
  { name: 'The Liquor Whiskers', avatar: '/avatars/cats/cat-wizard.jpg', color: '#98D8C8' },
  { name: 'Ray Pawston', avatar: '/avatars/cats/cat-zen.jpg', color: '#ABEBC6' },
]

const defaultCharacter: CharacterInfo = {
  name: 'Shadow Rickitty',
  avatar: '/avatars/cats/cat-ninja.jpg',
  color: '#45B7D1'
}

/**
 * Improved hash function using FNV-1a algorithm for better distribution
 */
function improvedHash(str: string): number {
  const FNV_OFFSET_BASIS = 2166136261
  const FNV_PRIME = 16777619

  let hash = FNV_OFFSET_BASIS

  for (let i = 0; i < str.length; i++) {
    hash ^= str.charCodeAt(i)
    hash = Math.imul(hash, FNV_PRIME)
  }

  return Math.abs(hash >>> 0)
}

/**
 * Get character avatar information for a session name or ID
 * Uses a hash to deterministically map session IDs to cat avatars
 */
export function useCharacterAvatar(sessionName?: string | null): CharacterInfo {
  if (!sessionName) {
    return defaultCharacter
  }

  const hash = improvedHash(sessionName)
  const index = hash % fallbackAvatars.length

  return fallbackAvatars[index]
}

/**
 * Get all available characters
 */
export function getAllCharacters(): CharacterInfo[] {
  return [...fallbackAvatars]
}

/**
 * Check if a session name has a mapped character
 */
export function hasCharacter(sessionName?: string | null): boolean {
  return !!sessionName
}
```

**Step 2: Verify no TypeScript errors**

```bash
cd internal/server/frontend && npx nuxi typecheck
```

**Step 3: Commit**

```bash
git add internal/server/frontend/app/composables/useCharacterAvatar.ts
git commit -m "feat: replace DiceBear fallback avatars with local cat avatars"
```

---

### Task 3: Rewrite Database Seeder — Cat Themes Instead of DiceBear

**Files:**
- Modify: `internal/database/seeder.go`

**Step 1: Rewrite SeedAvatarThemes and replace seedDiceBearTheme**

Replace everything from line 40 onwards with cat theme seeding:

```go
// SeedAvatarThemes seeds the database with builtin cat avatar themes
func (db *Database) SeedAvatarThemes() error {
	repo := NewRepository(db)

	// Check if themes already exist
	themes, err := repo.GetAvatarThemes()
	if err != nil {
		return fmt.Errorf("failed to check existing themes: %w", err)
	}

	// Cat theme names
	catThemeNames := map[string]bool{
		"Neon Cats": true,
		"Dali Cats": true,
		"Line Cats": true,
	}

	// Track existing and clean up legacy themes (DiceBear, South Park, etc.)
	existingThemes := make(map[string]bool)
	for _, theme := range themes {
		if !catThemeNames[theme.Name] {
			fmt.Printf("Removing legacy avatar theme: %s\n", theme.Name)
			if err := repo.DeleteAvatarTheme(theme.ID); err != nil {
				fmt.Printf("Warning: failed to remove legacy theme %s: %v\n", theme.Name, err)
			}
			continue
		}
		existingThemes[theme.Name] = true
	}

	// Seed cat themes
	catThemes := []string{"Neon Cats", "Dali Cats", "Line Cats"}
	for _, themeName := range catThemes {
		if !existingThemes[themeName] {
			if err := db.seedCatTheme(repo, themeName); err != nil {
				return err
			}
		}
	}

	return nil
}

// seedCatTheme seeds a cat avatar theme with local image paths
func (db *Database) seedCatTheme(repo *Repository, themeName string) error {
	// Map theme names to file prefixes and descriptions
	prefixMap := map[string]string{
		"Neon Cats": "cat",
		"Dali Cats": "cat-dali",
		"Line Cats": "cat-line",
	}

	descriptionMap := map[string]string{
		"Neon Cats": "Neon cyberpunk cat avatars with glowing portraits",
		"Dali Cats": "Surrealist Dali-inspired cat avatars",
		"Line Cats": "Minimal line art cat avatars",
	}

	prefix, ok := prefixMap[themeName]
	if !ok {
		return fmt.Errorf("unknown cat theme: %s", themeName)
	}

	// Create theme
	theme := &AvatarTheme{
		Name:        themeName,
		Description: descriptionMap[themeName],
		IsBuiltin:   true,
	}

	err := repo.CreateAvatarTheme(theme)
	if err != nil {
		return fmt.Errorf("failed to create %s theme: %w", themeName, err)
	}

	// Cat characters: slug, name, color
	type catChar struct {
		slug  string
		name  string
		color string
	}

	characters := []catChar{
		{"coder", "Bubbles de Catt", "#85C1E2"},
		{"coffee", "Julian Purresso", "#F8B88B"},
		{"detective", "Inspector Lahisker", "#F9E79F"},
		{"dj", "J-Roc Meowski", "#BB8FCE"},
		{"hacker", "Cyrus Rootpaw", "#4ECDC4"},
		{"music", "Conky Catsworth", "#F1948A"},
		{"ninja", "Shadow Rickitty", "#45B7D1"},
		{"pirate", "Captain Shittpaw", "#FF6B6B"},
		{"rocket", "Ricky Lawnchpurr", "#FFA07A"},
		{"sleeping", "Randy Catnaps", "#D7BDE2"},
		{"wizard", "The Liquor Whiskers", "#98D8C8"},
		{"zen", "Ray Pawston", "#ABEBC6"},
	}

	var avatarIDs []int64
	for _, ch := range characters {
		// Build filename: "cat-coder.jpg" or "cat-dali-coder.jpg"
		var filename string
		if prefix == "cat" {
			filename = fmt.Sprintf("%s-%s.jpg", prefix, ch.slug)
		} else {
			filename = fmt.Sprintf("%s-%s.jpg", prefix, ch.slug)
		}

		avatar := &Avatar{
			ThemeID:   theme.ID,
			Name:      ch.name,
			Type:      AvatarTypePreset,
			ImagePath: fmt.Sprintf("cats/%s", filename),
			Color:     ch.color,
		}

		err := repo.CreateAvatar(avatar)
		if err != nil {
			fmt.Printf("Warning: failed to seed avatar %s: %v\n", avatar.Name, err)
			continue
		}
		avatarIDs = append(avatarIDs, avatar.ID)
	}

	// Set a random avatar as the representative for the theme
	if len(avatarIDs) > 0 {
		randomIndex := rand.Intn(len(avatarIDs))
		theme.RepresentativeAvatarID = &avatarIDs[randomIndex]
		if err := repo.UpdateAvatarTheme(theme); err != nil {
			fmt.Printf("Warning: failed to set representative avatar for %s theme: %v\n", themeName, err)
		}
	}

	return nil
}
```

**Step 2: Run Go build to verify**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/database/seeder.go
git commit -m "feat: replace DiceBear themes with Neon/Dali/Line cat themes in seeder"
```

---

### Task 4: Remove DiceBear Type from Models

**Files:**
- Modify: `internal/database/models.go:252-267`
- Modify: `internal/database/schema.sql` (comments only)

**Step 1: Remove AvatarTypeDiceBear constant and update comments**

In `models.go`, remove the `AvatarTypeDiceBear` line and update comments:

```go
const (
	AvatarTypePreset      AvatarType = "preset"
	AvatarTypeAIGenerated AvatarType = "ai_generated"
)

// Avatar represents an individual avatar within a theme
type Avatar struct {
	ID        int64      `json:"id"`
	ThemeID   int64      `json:"theme_id"`
	Name      string     `json:"name"`
	Type      AvatarType `json:"type"` // 'preset' or 'ai_generated'
	ImagePath string     `json:"image_path,omitempty"`
	ImageURL  string     `json:"image_url,omitempty"`
	Style     string     `json:"style,omitempty"`
	Seed      string     `json:"seed,omitempty"`
	Color     string     `json:"color,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
```

In `schema.sql`, update comments to remove DiceBear references (lines ~186, 205-209).

**Step 2: Find and fix all Go references to AvatarTypeDiceBear**

Search for `AvatarTypeDiceBear` and `"dicebear"` in Go files:
- `internal/server/handlers/avatar_handler.go:276` — remove the DiceBear redirect block (lines 275-283)
- `internal/server/auth_handler.go:284` — change `avatar.Type == "dicebear"` to remove the DiceBear branch
- `internal/server/auth_handlers.go:469` — same change

For both `auth_handler.go` and `auth_handlers.go`, simplify the avatar image logic to:

```go
// Set avatar image
if avatar.ImagePath != "" {
    if !strings.HasPrefix(avatar.ImagePath, "http") && !strings.HasPrefix(avatar.ImagePath, "/") {
        response["avatar_image"] = fmt.Sprintf("/api/avatars/%d/image", avatar.ID)
    } else {
        response["avatar_image"] = avatar.ImagePath
    }
} else if avatar.ImageURL != "" {
    response["avatar_image"] = avatar.ImageURL
}
```

**Step 3: Remove HandleGenerateRandomAvatar handler**

In `avatar_handler.go`, remove the `HandleGenerateRandomAvatar` function (lines 325-349) and find/remove its route registration.

Check `internal/server/routes/avatar.go` for the route:

```bash
grep -n "generate-random\|GenerateRandom" internal/server/routes/avatar.go
```

Remove that route.

**Step 4: Build and verify**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add internal/database/models.go internal/database/schema.sql \
  internal/server/handlers/avatar_handler.go \
  internal/server/auth_handler.go internal/server/auth_handlers.go \
  internal/server/routes/avatar.go
git commit -m "refactor: remove DiceBear avatar type from backend"
```

---

### Task 5: Delete DiceBear Service Package

**Files:**
- Delete: `internal/avatars/dicebear_service.go`
- Delete: `internal/avatars/names.go`

**Step 1: Verify no Go imports reference internal/avatars**

```bash
grep -r '"github.com/schlunsen/wee-editor/internal/avatars"' --include="*.go" .
```

Expected: no results (already confirmed)

**Step 2: Delete the files**

```bash
rm internal/avatars/dicebear_service.go internal/avatars/names.go
```

Check if the directory is now empty and remove it:

```bash
ls internal/avatars/
rmdir internal/avatars/ 2>/dev/null || echo "Directory not empty, check remaining files"
```

**Step 3: Build**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add -A internal/avatars/
git commit -m "chore: remove DiceBear service package (replaced by local cat avatars)"
```

---

### Task 6: Clean Up Frontend — Remove DiceBear References

**Files:**
- Modify: `internal/server/frontend/app/composables/useAvatarThemes.ts`
- Modify: `internal/server/frontend/app/composables/useAvatarSelection.ts`
- Modify: `internal/server/frontend/app/components/avatars/AvatarPicker.vue`
- Modify: `internal/server/frontend/app/components/agents/SessionItem.vue`
- Modify: `internal/server/frontend/app/components/agents/MetricsSidebar.vue`
- Modify: `internal/server/frontend/app/pages/avatar-manager.vue`

**Step 1: useAvatarThemes.ts**

Line 14: change type from `'preset' | 'dicebear' | 'ai_generated'` to `'preset' | 'ai_generated'`

Lines 17-18: remove DiceBear style/seed comments

Lines 138-146: simplify `getAvatarImageUrl` — remove the DiceBear branch:

```typescript
export function getAvatarImageUrl(avatar: Avatar): string {
  // For avatars with a relative image path, use the API endpoint
  if (avatar.image_path && !avatar.image_path.startsWith('http') && !avatar.image_path.startsWith('/')) {
    return `/api/avatars/${avatar.id}/image`
  }

  // For avatars with absolute URLs or paths, use directly
  if (avatar.image_path) {
    return avatar.image_path
  }

  // Fallback to API endpoint for image serving
  return `/api/avatars/${avatar.id}/image`
}
```

**Step 2: useAvatarSelection.ts**

Lines 111-113: remove the DiceBear special case in `getAvatarImageSource`. Simplify to:

```typescript
export function getAvatarImageSource(avatar: Avatar | null, character: CharacterInfo | null): string {
  if (avatar) {
    // For preset avatars, construct URL from image path
    if (avatar.image_path) {
      if (!avatar.image_path.startsWith('http') && !avatar.image_path.startsWith('/')) {
        return `/api/avatars/${avatar.id}/image`
      }
      return avatar.image_path
    }
  }

  if (character) {
    return character.avatar
  }

  return '/avatars/cats/cat-ninja.jpg'
}
```

**Step 3: AvatarPicker.vue**

Lines 213-219: remove the `dicebearThemeMap` object entirely.

Lines 353-364: simplify `loadThemeIcons` — remove the DiceBear branch, use only the "fetch first avatar" path for all themes:

```typescript
const loadThemeIcons = async () => {
  const icons: Record<number, string> = {}

  for (const theme of themes.value) {
    // Use the first avatar as representative for all themes
    try {
      const avatars = await fetchThemeAvatars(theme.id)
      if (avatars.length > 0) {
```

(keep the rest of the existing non-DiceBear logic)

Lines 288-291: in the local `getAvatarImageUrl`, remove the DiceBear branch (same pattern as useAvatarThemes.ts).

**Step 4: SessionItem.vue**

Line 400: update comment from "Fall back to hash-based DiceBear character avatar" to "Fall back to hash-based cat avatar"

**Step 5: MetricsSidebar.vue**

Line 386: update comment from "Get effective avatar (persistent or hash-based DiceBear fallback)" to "Get effective avatar (persistent or hash-based cat fallback)"

Line 396: update comment from "Fall back to hash-based DiceBear character avatar" to "Fall back to hash-based cat avatar"

**Step 6: avatar-manager.vue**

Lines 240-242: remove `case 'dicebear': return 'DiceBear'` from `formatAvatarType`.

**Step 7: Verify frontend builds**

```bash
cd internal/server/frontend && npm run build
```

**Step 8: Commit**

```bash
git add internal/server/frontend/app/composables/useAvatarThemes.ts \
  internal/server/frontend/app/composables/useAvatarSelection.ts \
  internal/server/frontend/app/components/avatars/AvatarPicker.vue \
  internal/server/frontend/app/components/agents/SessionItem.vue \
  internal/server/frontend/app/components/agents/MetricsSidebar.vue \
  internal/server/frontend/app/pages/avatar-manager.vue
git commit -m "refactor: remove all DiceBear references from frontend"
```

---

### Task 7: Clean Up iOS DiceBear References

**Files:**
- Modify: `ios/wee/wee/Services/AvatarManager.swift` (lines ~86-88, 157-158)
- Modify: `ios/wee/wee/Services/WeeAPIClient.swift` (lines ~72-76, 953, 977)

**Step 1: AvatarManager.swift**

Remove the two DiceBear special-case blocks that download from external URLs. The `type == "dicebear"` branches should be removed — all avatars now use the standard API endpoint path.

**Step 2: WeeAPIClient.swift**

- Line 72: change type comment from `"preset", "dicebear", "ai_generated"` to `"preset", "ai_generated"`
- Lines 74-76: remove DiceBear-specific field comments
- Line 953: update comment
- Line 977: remove or update the "Download from external URL (e.g., DiceBear)" comment

**Step 3: Build iOS project**

```bash
xcodebuild -project ios/wee/wee.xcodeproj -scheme wee -configuration Debug -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | tail -5
```

**Step 4: Commit**

```bash
git add ios/wee/wee/Services/AvatarManager.swift ios/wee/wee/Services/WeeAPIClient.swift
git commit -m "refactor: remove DiceBear references from iOS app"
```

---

### Task 8: Clean Up Remaining DiceBear References

**Files:**
- Modify: `internal/server/agents/messages.go:263-267` — update type comment, remove DiceBear field comments
- Modify: `internal/server/frontend/app/stores/session/types.ts` — if it has dicebear type refs
- Modify: `internal/server/frontend/app/types/message.ts` — if it has dicebear type refs

**Step 1: Search for any remaining "dicebear" or "DiceBear" references**

```bash
grep -ri "dicebear\|dice.bear" --include="*.go" --include="*.ts" --include="*.vue" --include="*.swift" . | grep -v node_modules | grep -v CHANGELOG | grep -v docs/plans
```

Fix every remaining reference found.

**Step 2: Build everything**

```bash
go build ./...
cd internal/server/frontend && npm run build
```

**Step 3: Commit**

```bash
git add -A
git commit -m "chore: remove all remaining DiceBear references"
```

---

### Task 9: Verify End-to-End

**Step 1: Delete existing database to force re-seed**

```bash
rm ~/.claude/wee/wee.db 2>/dev/null
```

**Step 2: Start the server**

```bash
go run ./cmd/wee --analytics
```

**Step 3: Verify cat themes are seeded**

```bash
API_KEY=$(cat ~/.claude/analytics/.secret)
curl -k https://localhost:3333/api/avatars/themes | jq '.themes[] | {name, avatar_count}'
```

Expected:
```json
{"name": "Neon Cats", "avatar_count": 12}
{"name": "Dali Cats", "avatar_count": 12}
{"name": "Line Cats", "avatar_count": 12}
```

**Step 4: Verify avatar images load**

```bash
curl -k https://localhost:3333/api/avatars/themes/1 | jq '.avatars[0]'
```

Verify `type` is `"preset"` and `image_path` starts with `"cats/"`.

**Step 5: Open dashboard and check avatars render**

Open `https://localhost:3333` — verify session avatars show cat images, avatar picker shows 3 cat themes.
