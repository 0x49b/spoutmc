package permissions

// Definition is a default permission row used only to seed an empty or outdated database.
// The runtime source of truth is the `permissions` table; you can add/edit/remove rows via API (admin) or SQL.
type Definition struct {
	Key         string
	Description string
}

// Definitions are default seeds applied on startup (insert missing keys; sync description from defaults when unchanged).
// They are not used for authorization logic—see AllKeysFromDB.
var Definitions = []Definition{
	{Key: "auth.user.manage", Description: "Manage users and roles in the configuration area"},
	{Key: "auth.user.create", Description: "Create user accounts"},
	{Key: "server.list.read", Description: "View servers, infrastructure, and plugins"},
	{Key: "server.manage", Description: "Create, edit, and control server instances"},
	{Key: "player.list.read", Description: "View online and banned players"},
	{Key: "player.manage", Description: "Kick, ban, and message players"},
}
