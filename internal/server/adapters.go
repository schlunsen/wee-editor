package server

import "github.com/schlunsen/wee-editor/internal/server/handlers"

// userStoreAdapter adapts the UserStore to implement handlers.UserStoreInterface
type userStoreAdapter struct {
	store *UserStore
}

// GetUser gets a user from the store
func (a *userStoreAdapter) GetUser(username string) (handlers.UserInterface, bool) {
	if a.store == nil {
		return nil, false
	}
	user, ok := a.store.users[username]
	if !ok {
		return nil, false
	}
	return &userAdapter{user: user}, true
}

// userAdapter adapts the User struct to implement handlers.UserInterface
type userAdapter struct {
	user *User
}

// SetAvatarID sets the user's avatar ID
func (a *userAdapter) SetAvatarID(avatarID *int64) {
	if a.user != nil {
		a.user.AvatarID = avatarID
	}
}
