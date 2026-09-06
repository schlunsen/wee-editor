package database

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

// SeedDefaultProject seeds a default "app" project at /app if no projects exist.
// This ensures sandbox instances have a ready-to-use project on first boot.
func (db *Database) SeedDefaultProject() error {
	projectRepo := NewProjectRepository(db)

	// Check if any projects already exist — don't overwrite user setup
	projects, err := projectRepo.GetAllProjects()
	if err != nil {
		return fmt.Errorf("failed to check existing projects: %w", err)
	}
	if len(projects) > 0 {
		return nil // User already has projects, skip seeding
	}

	project := &Project{
		ID:       uuid.New().String(),
		Name:     "app",
		Path:     "/app",
		IsActive: true,
		Color:    "#8b5cf6",
	}

	if err := projectRepo.CreateProject(project); err != nil {
		return fmt.Errorf("failed to seed default project: %w", err)
	}

	fmt.Println("Seeded default project: app (/app)")
	return nil
}

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

	// Track existing and clean up legacy themes
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
		filename := fmt.Sprintf("%s-%s.jpg", prefix, ch.slug)

		avatar := &Avatar{
			ThemeID:   theme.ID,
			Name:      ch.name,
			Type:      AvatarTypePreset,
			ImagePath: fmt.Sprintf("/avatars/cats/%s", filename),
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
