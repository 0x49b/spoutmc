package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type PropertiesFile struct {
	Lines []string
	Map   map[string]string
}

func main() {
	filePath := "server.properties"

	// Read and parse the file
	config, err := readProperties(filePath)
	if err != nil {
		fmt.Println("Error reading properties:", err)
		return
	}

	// Example: Change some settings
	config.Map["max-players"] = "50"
	config.Map["motd"] = "Welcome to my Go-powered Minecraft server!"

	// Write back to file
	if err := writeProperties(filePath, config); err != nil {
		fmt.Println("Error writing properties:", err)
	}
}

// readProperties reads and parses the server.properties file
func readProperties(filePath string) (*PropertiesFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &PropertiesFile{
		Lines: []string{},
		Map:   make(map[string]string),
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		config.Lines = append(config.Lines, line)

		// Ignore comments or empty lines
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle key=value# or key=value (optional stray hash)
		if idx := strings.Index(line, "="); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(strings.TrimRight(line[idx+1:], "#"))
			config.Map[key] = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// writeProperties rewrites the file with updated values
func writeProperties(filePath string, config *PropertiesFile) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range config.Lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			writer.WriteString(line + "\n")
			continue
		}

		// Update value from map
		if idx := strings.Index(line, "="); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			if val, ok := config.Map[key]; ok {
				writer.WriteString(fmt.Sprintf("%s=%s#\n", key, val))
			} else {
				writer.WriteString(line + "\n") // keep original
			}
		}
	}

	return writer.Flush()
}
