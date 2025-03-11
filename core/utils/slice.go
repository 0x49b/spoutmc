package utils

// CheckInStringSlice Check if the string is in the slice
func CheckInStringSlice(stringSlice []string, targetString string) bool {
	for _, str := range stringSlice {
		if str == targetString {
			return true
		}
	}
	return false
}
