package access

type Definition struct {
	Key         string
	Description string
}

var Definitions = []Definition{
	{Key: "auth.user.manage", Description: "Manage users and roles in the configuration area"},
	{Key: "auth.user.create", Description: "Create user accounts"},
	{Key: "server.list.read", Description: "View servers, infrastructure, and plugins"},
	{Key: "server.manage", Description: "Create, edit, and control server instances"},
	{Key: "player.list.read", Description: "View online and banned players"},
	{Key: "player.manage", Description: "Kick, ban, and message players"},
	{Key: "player.conversations.view_all", Description: "View all staff↔player conversations"},
	{Key: "plugins.manage", Description: "Manage plugin registry and server assignments"},
}
