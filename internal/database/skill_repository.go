package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SkillRepository provides data access methods for skills
type SkillRepository struct {
	db *Database
}

// NewSkillRepository creates a new skill repository instance
func NewSkillRepository(db *Database) *SkillRepository {
	return &SkillRepository{db: db}
}

// GetAllSkills retrieves all skills, optionally filtered by scope
func (r *SkillRepository) GetAllSkills(scope string) ([]*Skill, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	var query string
	var args []interface{}

	if scope != "" {
		query = `
			SELECT id, name, description, scope, path, frontmatter_json, body,
				user_invocable, disable_model_invocation, allowed_tools,
				model, effort, context, agent, argument_hint, shell,
				created_at, updated_at
			FROM skills
			WHERE scope = ?
			ORDER BY name ASC
		`
		args = append(args, scope)
	} else {
		query = `
			SELECT id, name, description, scope, path, frontmatter_json, body,
				user_invocable, disable_model_invocation, allowed_tools,
				model, effort, context, agent, argument_hint, shell,
				created_at, updated_at
			FROM skills
			ORDER BY scope ASC, name ASC
		`
	}

	rows, err := r.db.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query skills: %w", err)
	}
	defer rows.Close()

	var skills []*Skill
	for rows.Next() {
		s := &Skill{}
		var effort, context, shell sql.NullString
		err := rows.Scan(
			&s.ID, &s.Name, &s.Description, &s.Scope, &s.Path,
			&s.FrontmatterJSON, &s.Body,
			&s.UserInvocable, &s.DisableModelInvocation, &s.AllowedTools,
			&s.Model, &effort, &context, &s.Agent, &s.ArgumentHint, &shell,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan skill: %w", err)
		}
		s.Effort = effort.String
		s.Context = context.String
		s.Shell = shell.String
		skills = append(skills, s)
	}

	return skills, rows.Err()
}

// GetSkillByName retrieves a single skill by name
func (r *SkillRepository) GetSkillByName(name string) (*Skill, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, name, description, scope, path, frontmatter_json, body,
			user_invocable, disable_model_invocation, allowed_tools,
			model, effort, context, agent, argument_hint, shell,
			created_at, updated_at
		FROM skills
		WHERE name = ?
	`

	s := &Skill{}
	var effort, context, shell sql.NullString
	err := r.db.db.QueryRow(query, name).Scan(
		&s.ID, &s.Name, &s.Description, &s.Scope, &s.Path,
		&s.FrontmatterJSON, &s.Body,
		&s.UserInvocable, &s.DisableModelInvocation, &s.AllowedTools,
		&s.Model, &effort, &context, &s.Agent, &s.ArgumentHint, &shell,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}
	s.Effort = effort.String
	s.Context = context.String
	s.Shell = shell.String

	return s, nil
}

// GetSkillByID retrieves a single skill by ID
func (r *SkillRepository) GetSkillByID(id string) (*Skill, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, name, description, scope, path, frontmatter_json, body,
			user_invocable, disable_model_invocation, allowed_tools,
			model, effort, context, agent, argument_hint, shell,
			created_at, updated_at
		FROM skills
		WHERE id = ?
	`

	s := &Skill{}
	var effort, context, shell sql.NullString
	err := r.db.db.QueryRow(query, id).Scan(
		&s.ID, &s.Name, &s.Description, &s.Scope, &s.Path,
		&s.FrontmatterJSON, &s.Body,
		&s.UserInvocable, &s.DisableModelInvocation, &s.AllowedTools,
		&s.Model, &effort, &context, &s.Agent, &s.ArgumentHint, &shell,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get skill by ID: %w", err)
	}
	s.Effort = effort.String
	s.Context = context.String
	s.Shell = shell.String

	return s, nil
}

// SaveSkill creates or updates a skill in the database
func (r *SkillRepository) SaveSkill(skill *Skill) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	if skill.ID == "" {
		skill.ID = uuid.New().String()
	}

	now := time.Now()

	// Upsert by name
	query := `
		INSERT INTO skills (
			id, name, description, scope, path, frontmatter_json, body,
			user_invocable, disable_model_invocation, allowed_tools,
			model, effort, context, agent, argument_hint, shell,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			description = excluded.description,
			scope = excluded.scope,
			path = excluded.path,
			frontmatter_json = excluded.frontmatter_json,
			body = excluded.body,
			user_invocable = excluded.user_invocable,
			disable_model_invocation = excluded.disable_model_invocation,
			allowed_tools = excluded.allowed_tools,
			model = excluded.model,
			effort = excluded.effort,
			context = excluded.context,
			agent = excluded.agent,
			argument_hint = excluded.argument_hint,
			shell = excluded.shell,
			updated_at = excluded.updated_at
	`

	// Convert empty strings to nil for columns with CHECK constraints
	// SQLite CHECK(col IS NULL OR col IN (...)) requires NULL, not empty string
	nilIfEmpty := func(s string) interface{} {
		if s == "" {
			return nil
		}
		return s
	}

	_, err := r.db.db.Exec(query,
		skill.ID, skill.Name, skill.Description, skill.Scope, skill.Path,
		skill.FrontmatterJSON, skill.Body,
		skill.UserInvocable, skill.DisableModelInvocation, skill.AllowedTools,
		skill.Model, nilIfEmpty(skill.Effort), nilIfEmpty(skill.Context),
		skill.Agent, skill.ArgumentHint, nilIfEmpty(skill.Shell),
		now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to save skill: %w", err)
	}

	return nil
}

// DeleteSkill removes a skill by name
func (r *SkillRepository) DeleteSkill(name string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	result, err := r.db.db.Exec("DELETE FROM skills WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("skill not found: %s", name)
	}

	return nil
}

// DeleteSkillsByScope removes all skills for a given scope
func (r *SkillRepository) DeleteSkillsByScope(scope string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	_, err := r.db.db.Exec("DELETE FROM skills WHERE scope = ?", scope)
	if err != nil {
		return fmt.Errorf("failed to delete skills by scope: %w", err)
	}

	return nil
}
