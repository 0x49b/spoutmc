package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

func parser() {
	// Load the TOML file
	filePath := "velocity.toml"
	configData, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Parse the TOML
	tree, err := toml.Load(string(configData))
	if err != nil {
		fmt.Println("Error parsing TOML:", err)
		return
	}

	// Modify some values as examples
	tree.Set("motd", "<#ff0000>Modified Velocity Server")
	tree.Set("online-mode", false)
	tree.Set("servers.lobby", "192.168.1.100:25566") // example IP change

	// Serialize back to TOML format
	modifiedConfig := tree.String()

	// Write back to file
	err = os.WriteFile(filePath, []byte(modifiedConfig), 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	fmt.Println("Configuration file successfully modified.")
}
