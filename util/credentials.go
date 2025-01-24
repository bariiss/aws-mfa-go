package util

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"golang.org/x/term"
	"log"
	"os"
	"path/filepath"
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

	return time.Time{}, fmt.Errorf("\nExpiration time not found for profile %s", profile)
}

func CheckProfile(awsProfile string, awsRegion string) bool {
	if awsProfile == "" || awsRegion == "" {
		color.Red("🚨 Missing environment variables. AWS_MFA_GO_USER and AWS_MFA_GO_REGION are required.")
		color.Yellow("👉 Set these variables to continue. Example:")
		color.Yellow("   export AWS_MFA_GO_USER=<profile>")
		color.Yellow("   export AWS_MFA_GO_REGION=<region>")
		color.Yellow("\n👉 Save your AWS credentials in the credentials file with a -go suffix:\n   [<profile>-go]\n   aws_access_key_id = <aws_access_key_id>\n   aws_secret_access_key = <aws_secret_access_key>\n   aws_mfa_device = <aws_mfa_device>\n   aws_mfa_duration = <aws_mfa_duration>\n   aws_mfa_secret_key = <aws_mfa_secret_key>")
		return true
	}
	return false
}

func GetCredentialsFilePath() string {
	homeDirs, err := os.UserHomeDir()

	if err != nil {
		log.Fatalf("\nError getting home directory: %s", err)
	}

	return filepath.Join(homeDirs, ".aws", "credentials")
}

func ExpirationTimeValid(awsProfile string, expirationTime time.Time) bool {
	if expirationTime.IsZero() {
		fmt.Printf("\nProfile '%s' does not exist, skipping expiration check.\n", awsProfile)
		return false
	}

	if time.Now().Before(expirationTime) {
		remainingDuration := expirationTime.Sub(time.Now().UTC()).Hours()
		fmt.Printf("📌 The current token for profile '%s' is valid until %v (about %.2f hours remaining).\n", awsProfile, expirationTime.Format("2006-01-02 15:04:05"), remainingDuration)
		return true
	}

	return false
}

func ConfirmContinuation() bool {
	fmt.Print("🤔 Do you want to continue and generate a new token? (y/n): ")

	// Set the terminal to raw mode to capture a single keystroke
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))

	if err != nil {
		fmt.Printf("\nError setting terminal to raw mode: %s\n", err)
		return false
	}

	defer func(fd int, oldState *term.State) {
		err := term.Restore(fd, oldState)
		if err != nil {
			fmt.Printf("\nError restoring terminal state: %s\n", err)
		}
	}(int(os.Stdin.Fd()), oldState)

	// Read a single character
	var response = make([]byte, 1)
	_, err = os.Stdin.Read(response)

	if err != nil {
		fmt.Printf("\nError reading response: %s\n", err)
		return false
	}

	// Check if the response is "y"
	return response[0] == 'y'
}

func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("\nError closing file: %s", err)
		}
	}(file)

	// Pre-allocate slice with an initial capacity
	lines := make([]string, 0, 50)
	scanner := bufio.NewScanner(file)

	// Optimize scanner buffer for larger files
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

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
	// Pre-allocate slice with exact size needed
	result := make([]string, 9)
	result[0] = profileHeader
	result[1] = "aws_access_key_id = " + c.AccessKeyID
	result[2] = "aws_secret_access_key = " + c.SecretAccessKey
	result[3] = "aws_mfa_device = " + c.MFADevice
	result[4] = "aws_mfa_duration = " + c.MFADuration
	result[5] = "aws_mfa_secret_key = " + c.MFASecretKey
	result[6] = "assumed_role = " + strconv.FormatBool(c.AssumedRole)
	result[7] = "aws_session_token = " + c.SessionToken
	result[8] = "expiration = " + c.Expiration.Format("2006-01-02 15:04:05")
	return result
}

func writeLinesToFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("\nError closing file: %s", err)
		}
	}(file)

	// Use bufio.Writer with larger buffer
	writer := bufio.NewWriterSize(file, 64*1024)
	defer func(writer *bufio.Writer) {
		err := writer.Flush()
		if err != nil {
			fmt.Printf("\nError flushing buffer: %s", err)
		}
	}(writer)

	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return nil
}
