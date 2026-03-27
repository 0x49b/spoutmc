package access

import "spoutmc/internal/models"

func BuildUserResponse(user *models.User) models.UserResponse {
	if user == nil {
		return models.UserResponse{}
	}
	perms := EffectivePermissionKeysFromUser(user)
	roles := make([]models.RoleResponse, len(user.Roles))
	for i, r := range user.Roles {
		roles[i] = models.RoleResponse{
			ID:          r.ID,
			Name:        r.Name,
			DisplayName: r.DisplayName,
			Slug:        r.Slug,
		}
	}
	direct := make([]models.PermissionResponse, len(user.DirectPermissions))
	for i, p := range user.DirectPermissions {
		direct[i] = models.PermissionResponse{ID: p.ID, Key: p.Key, Description: p.Description}
	}
	return models.UserResponse{
		ID:                user.ID,
		CreatedAt:         user.CreatedAt,
		MinecraftID:       user.MinecraftID,
		MinecraftName:     user.MinecraftName,
		DisplayName:       user.DisplayName,
		Email:             user.Email,
		Roles:             roles,
		Permissions:       perms,
		DirectPermissions: direct,
		Avatar:            user.Avatar,
	}
}
