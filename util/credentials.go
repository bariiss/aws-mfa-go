package util

import (
	"bufio"
	"fmt"
	"log"
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

	profileIndex, profileExists := findProfileIndex(lines, p)

	updatedProfileLines := formatProfileData(*c)

	if profileExists {
		lines = updateProfileLines(lines, profileIndex, updatedProfileLines)
	} else {
		lines = append(lines, "")
		lines = append(lines, "["+p+"]")
		lines = append(lines, updatedProfileLines...)
	}

	return writeLinesToFile(f, lines)
}

func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("Error closing file: %s", err)
		}
	}(file)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func findProfileIndex(lines []string, profile string) (int, bool) {
	profileHeader := "[" + profile + "]"
	for i, line := range lines {
		if strings.TrimSpace(line) == profileHeader {
			return i, true
		}
	}
	return -1, false
}

func updateProfileLines(lines []string, profileIndex int, updatedProfileLines []string) []string {
	updatedLines := make([]string, 0, len(lines))

	updatedLines = append(updatedLines, lines[:profileIndex]...)

	updatedLines = append(updatedLines, lines[profileIndex])

	updatedLines = append(updatedLines, updatedProfileLines...)

	nextProfileIndex := findNextProfileIndex(lines, profileIndex)
	if nextProfileIndex < len(lines) {
		updatedLines = append(updatedLines, lines[nextProfileIndex:]...)
	}

	return updatedLines
}

func findNextProfileIndex(lines []string, currentIndex int) int {
	for i := currentIndex; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "[") && i != currentIndex {
			return i
		}
	}
	return len(lines)
}

func formatProfileData(c Credentials) []string {
	return []string{
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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("Error closing file: %s", err)
		}
	}(file)

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}

func ReadExpirationTime(filePath, profile string) (time.Time, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return time.Time{}, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("Error closing file: %s", err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	profileHeader := "[" + profile + "]"
	inProfile := false

	for scanner.Scan() {
		line := scanner.Text()
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

	if err := scanner.Err(); err != nil {
		return time.Time{}, err
	}

	return time.Time{}, fmt.Errorf("expiration time not found for profile %s", profile)
}
