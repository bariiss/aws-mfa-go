package util

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func SaveCredentialsToFile(f string, p string, c *Credentials) error {
	lines, err := readLines(f)
	if err != nil {
		return err
	}

	profileHeader := "[" + p + "]"
	profileIndex, profileExists := findProfileIndex(lines, profileHeader)

	updatedProfileLines := formatProfileData(*c, profileHeader)

	if profileExists {
		lines = updateProfileLines(lines, profileIndex, updatedProfileLines)
	} else {
		lines = append(lines, append([]string{""}, updatedProfileLines...)...)
	}

	return writeLinesToFile(f, lines)
}

func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func findProfileIndex(lines []string, profileHeader string) (int, bool) {
	for i, line := range lines {
		if strings.TrimSpace(line) == profileHeader {
			return i, true
		}
	}
	return -1, false
}

func updateProfileLines(lines []string, profileIndex int, updatedProfileLines []string) []string {
	nextProfileIndex := findNextProfileIndex(lines, profileIndex)
	return append(append(lines[:profileIndex], updatedProfileLines...), lines[nextProfileIndex:]...)
}

func findNextProfileIndex(lines []string, currentIndex int) int {
	for i := currentIndex + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "[") {
			return i
		}
	}
	return len(lines)
}

func formatProfileData(c Credentials, profileHeader string) []string {
	return []string{
		profileHeader,
		"aws_access_key_id = " + c.AccessKeyID,
		"aws_secret_access_key = " + c.SecretAccessKey,
		"aws_mfa_device = " + c.MFADevice,
		"aws_mfa_duration = " + c.MFADuration,
		"aws_mfa_secret_key = " + c.MFASecretKey,
		"assumed_role = " + strconv.FormatBool(c.AssumedRole),
		"aws_session_token = " + c.SessionToken,
		"expiration = " + c.Expiration.Format("2006-01-02 15:04:05"),
	}
}

func writeLinesToFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func ReadExpirationTime(filePath, profile string) (time.Time, error) {
	lines, err := readLines(filePath)
	if err != nil {
		return time.Time{}, err
	}

	profileHeader := "[" + profile + "]"
	inProfile := false

	for _, line := range lines {
		if strings.TrimSpace(line) == profileHeader {
			inProfile = true
			continue
		}
		if inProfile && strings.HasPrefix(line, "[") {
			break
		}
		if inProfile && strings.Contains(line, "expiration =") {
			expirationStr := strings.SplitN(line, "=", 2)[1]
			expirationStr = strings.TrimSpace(expirationStr)
			return time.Parse("2006-01-02 15:04:05", expirationStr)
		}
	}

	return time.Time{}, fmt.Errorf("expiration time not found for profile %s", profile)
}
