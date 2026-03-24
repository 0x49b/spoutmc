package access

// RolePermissionKeys maps default role names to permission keys (seeded on startup).
// Admin is handled separately (all keys).
var RolePermissionKeys = map[string][]string{
	"manager": {
		"auth.user.manage",
		"auth.user.create",
		"server.list.read",
		"server.manage",
		"player.list.read",
		"player.manage",
		"plugins.manage",
	},
	"editor": {
		"server.list.read",
		"server.manage",
	},
	"mod": {
		"server.list.read",
		"player.list.read",
		"player.manage",
	},
	"support": {
		"server.list.read",
		"player.list.read",
	},
}
