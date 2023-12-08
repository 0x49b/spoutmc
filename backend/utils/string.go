package utils

import "math/rand"

func RandomString(length int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ#!$_")
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// Remove Item from slice
func Remove(slice []string, itemToRemove string) []string {
	var updatedSlice []string

	for _, item := range slice {
		if item != itemToRemove {
			updatedSlice = append(updatedSlice, item)
		}
	}

	return updatedSlice
}
