package adapters

import "github.com/schlunsen/wee-editor/internal/server/handlers"

// UserStoreAdapter adapts the UserStore to implement handlers.UserStoreInterface
type UserStoreAdapter struct {
	store *userStoreType
}

// NewUserStoreAdapter creates a new user store adapter
func NewUserStoreAdapter(store *userStoreType) *UserStoreAdapter {
	return &UserStoreAdapter{store: store}
}

// GetUser gets a user from the store
func (a *UserStoreAdapter) GetUser(username string) (handlers.UserInterface, bool) {
	if a.store == nil {
		return nil, false
	}
	user, ok := a.store.users[username]
	if !ok {
		return nil, false
	}
	return &UserAdapter{user: user}, true
}

// UserAdapter adapts the User struct to implement handlers.UserInterface
type UserAdapter struct {
	user *userType
}

// SetAvatarID sets the user's avatar ID
func (a *UserAdapter) SetAvatarID(avatarID *int64) {
	if a.user != nil {
		a.user.AvatarID = avatarID
	}
}

// Placeholder types for import - these will be replaced by the actual imports from server package
// The adapters package is separate, so we need to use unexported types or refactor further
// For now, we'll use interface{} to avoid circular imports

// userStoreType is a placeholder for the UserStore type from server package
type userStoreType struct {
	users map[string]*userType
}

// userType is a placeholder for the User type from server package
type userType struct {
	Username string
	AvatarID *int64
}
