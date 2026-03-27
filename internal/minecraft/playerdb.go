package minecraft

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"spoutmc/internal/models"

	"github.com/google/uuid"
)

func GetPlayerProfile(identifier string) (playerUUID uuid.UUID, username string, skinURL string, err error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return uuid.Nil, "", "", fmt.Errorf("identifier is empty")
	}

	u := url.URL{
		Scheme: "https",
		Host:   "playerdb.co",
		Path:   "/api/player/minecraft/" + url.PathEscape(identifier),
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return uuid.Nil, "", "", fmt.Errorf("error making GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return uuid.Nil, "", "", fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return uuid.Nil, "", "", fmt.Errorf("error reading response body: %w", err)
	}

	var data models.MojangResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return uuid.Nil, "", "", fmt.Errorf("error parsing JSON: %w", err)
	}

	if !data.Success {
		return uuid.Nil, "", "", fmt.Errorf("failed to resolve player %s: %s", identifier, data.Message)
	}

	playerUUID, err = uuid.Parse(data.Data.Player.RawID)
	if err != nil {
		return uuid.Nil, "", "", fmt.Errorf("error parsing UUID: %w", err)
	}

	return playerUUID, data.Data.Player.Username, data.Data.Player.SkinTexture, nil
}
