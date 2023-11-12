package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/fatih/color"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pquerna/otp/totp"
)

type (
	Credentials struct {
		AccessKeyID     string
		SecretAccessKey string
		MFADevice       string
		MFADuration     string
		MFASecretKey    string
		AssumedRole     bool
		SessionToken    string
		Expiration      time.Time
	}
	MFADetails struct {
		Device    string
		Duration  string
		SecretKey string
	}
)

var (
	awsProfile, awsRegion = os.Getenv("AWS_MFA_GO_USER"), os.Getenv("AWS_MFA_GO_REGION")
)

func main() {
	if _, err := exec.LookPath("aws"); err != nil {
		info := color.New(color.FgYellow).SprintFunc()
		fmt.Println(info("💩 AWS CLI is not installed. Please install it to use this program effectively." +
			"\n👉 https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html"))
	}

	if awsProfile == "" || awsRegion == "" {
		red := color.New(color.FgRed).SprintFunc()
		info := color.New(color.FgYellow).SprintFunc()
		fmt.Println(red("⚠️ The environment variables AWS_MFA_GO_USER and AWS_MFA_GO_REGION are not set."))
		fmt.Println("👉 Set these variables to continue. Example:")
		fmt.Println(info("   export AWS_MFA_GO_USER=<profile>"))
		fmt.Println(info("   export AWS_MFA_GO_REGION=<region>"))

		fmt.Println("👉 Save your AWS credentials in the credentials file in this format (with -go suffix):")
		fmt.Println(info("   [<profile>-go]"))
		fmt.Println(info("   aws_access_key_id = <aws_access_key_id>"))
		fmt.Println(info("   aws_secret_access_key = <aws_secret_access_key>"))
		fmt.Println(info("   aws_mfa_device = <aws_mfa_device>"))
		fmt.Println(info("   aws_mfa_duration = <aws_mfa_duration>"))
		fmt.Println(info("   aws_mfa_secret_key = <aws_mfa_secret_key>"))

		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(awsProfile+"-go"),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS config: %s", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %s\n", err)
		return
	}
	credentialsFilePath := filepath.Join(homeDir, ".aws", "credentials")

	expirationTime, err := readExpirationTime(credentialsFilePath, awsProfile)
	if err != nil {
		// if the profile does not exist, the expiration time is not found
	}

	if expirationTime.IsZero() {
		fmt.Printf("Profile '%s' does not exist, skipping expiration check.\n", awsProfile)
	}

	if time.Now().Before(expirationTime) {
		remainingDuration := expirationTime.Sub(time.Now())
		hoursRemaining := remainingDuration.Hours()
		fmt.Printf("📌 The current token for profile '%s' is still valid until %v (about %.2f hours remaining). Do you want to continue? (y/n): ", awsProfile, expirationTime.Format("2006-01-02 15:04:05"), hoursRemaining)

		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			fmt.Printf("Error reading response: %s\n", err)
			return
		}
		if response != "y" {
			fmt.Println("🤖 Operation aborted.")
			return
		}
	}

	mfaDetails, err := readMFADetails(credentialsFilePath, awsProfile+"-go")
	if err != nil {
		fmt.Printf("Error reading MFA details: %s\n", err)
		return
	}

	stsClient := sts.NewFromConfig(cfg)

	mfaToken, err := generateMFAToken(mfaDetails.SecretKey)
	if err != nil {
		log.Fatalf("Error generating MFA token: %s", err)
	}

	input := &sts.GetSessionTokenInput{
		SerialNumber: aws.String(mfaDetails.Device),
		TokenCode:    aws.String(mfaToken),
		DurationSeconds: func() *int32 {
			if mfaDetails.Duration != "" {
				duration, err := time.ParseDuration(mfaDetails.Duration)
				if err != nil {
					log.Fatalf("Error parsing duration: %s", err)
				}
				return aws.Int32(int32(duration.Seconds()))
			}
			return nil
		}(),
	}
	resp, err := stsClient.GetSessionToken(context.Background(), input)
	if err != nil {
		log.Fatalf("Error getting session token: %s", err)
	}

	// fill credentials struct with response
	c := Credentials{
		AccessKeyID:     *resp.Credentials.AccessKeyId,
		SecretAccessKey: *resp.Credentials.SecretAccessKey,
		MFADevice:       mfaDetails.Device,
		MFADuration:     mfaDetails.Duration,
		MFASecretKey:    mfaDetails.SecretKey,
		AssumedRole:     false,
		SessionToken:    *resp.Credentials.SessionToken,
		Expiration:      *resp.Credentials.Expiration,
	}

	// save credentials to file as profile without -go suffix

	err = saveCredentialsToFile(credentialsFilePath, awsProfile, c)
	if err != nil {
		log.Fatalf("Error saving credentials to file: %s", err)
	}

	fmt.Printf("👍 Credentials saved to file for profile '%s'.\n", awsProfile)
	fmt.Printf("🎉 You can now use profile '%s' with the AWS CLI.\n", awsProfile)
	fmt.Println("👉 Please check your AWS CLI config file to make sure you are using the correct profile name.")
	info := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("👉 Example: %s\n", info("aws configure --profile "+awsProfile))

	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println("👉 You can also use the profile with the AWS CLI directly.")
	fmt.Printf("👌 Example: %s\n", cyan("aws --profile "+awsProfile+" s3 ls"))
}

func generateMFAToken(secret string) (string, error) {
	token, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", fmt.Errorf("error generating TOTP code: %w", err)
	}
	return token, nil
}

func readMFADetails(filePath, profile string) (MFADetails, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return MFADetails{}, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("Error closing file: %s", err)
		}
	}(file)

	var details MFADetails
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
		if inProfile {
			keyValue := strings.SplitN(line, "=", 2)
			if len(keyValue) == 2 {
				key := strings.TrimSpace(keyValue[0])
				value := strings.TrimSpace(keyValue[1])

				switch key {
				case "aws_mfa_device":
					details.Device = value
				case "aws_mfa_duration":
					durationSeconds, err := strconv.ParseInt(value, 10, 64)
					if err != nil {
						return MFADetails{}, fmt.Errorf("error parsing duration: %w", err)
					}
					details.Duration = time.Duration(durationSeconds * int64(time.Second)).String()
				case "aws_mfa_secret_key":
					details.SecretKey = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return MFADetails{}, err
	}

	return details, nil
}

func saveCredentialsToFile(f string, p string, c Credentials) error {
	lines, err := readLines(f)
	if err != nil {
		return err
	}

	profileIndex, profileExists := findProfileIndex(lines, p)

	updatedProfileLines := formatProfileData(c)

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

func readExpirationTime(filePath, profile string) (time.Time, error) {
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
