package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/fatih/color"
	"golang.org/x/term"

	u "github.com/bariiss/aws-mfa-go/util"
)

var (
	awsProfile = os.Getenv("AWS_MFA_GO_USER")
	awsRegion  = os.Getenv("AWS_MFA_GO_REGION")
)

func main() {
	if checkProfile() {
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(awsProfile+"-go"),
		config.WithRegion(awsRegion),
	)

	if err != nil {
		log.Fatalf("\nUnable to load AWS config: %s", err)
	}

	credentialsFilePath := getCredentialsFilePath()
	expirationTime, err := u.ReadExpirationTime(credentialsFilePath, awsProfile)

	if err != nil {
		log.Printf("\nError reading expiration time: %s", err)
	}

	if expirationTimeValid(expirationTime) {
		if !confirmContinuation() {
			fmt.Println("\n🤖 Operation aborted.")
			return
		}
	}

	mfaDetails, err := u.ReadMFADetails(credentialsFilePath, awsProfile+"-go")

	if err != nil {
		log.Fatalf("\nError reading MFA details: %s", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	mfaToken, err := u.GenerateMFAToken(mfaDetails.SecretKey)

	if err != nil {
		log.Fatalf("\nError generating MFA token: %s", err)
	}

	resp, err := stsClient.GetSessionToken(context.Background(), &sts.GetSessionTokenInput{
		SerialNumber: aws.String(mfaDetails.Device),
		TokenCode:    aws.String(mfaToken),
		DurationSeconds: func() *int32 {
			if mfaDetails.Duration != "" {
				duration, _ := time.ParseDuration(mfaDetails.Duration)
				return aws.Int32(int32(duration.Seconds()))
			}
			return nil
		}(),
	})

	if err != nil {
		log.Fatalf("\nError getting session token: %s", err)
	}

	c := &u.Credentials{
		AccessKeyID:     *resp.Credentials.AccessKeyId,
		SecretAccessKey: *resp.Credentials.SecretAccessKey,
		MFADevice:       mfaDetails.Device,
		MFADuration:     mfaDetails.Duration,
		MFASecretKey:    mfaDetails.SecretKey,
		AssumedRole:     false,
		SessionToken:    *resp.Credentials.SessionToken,
		Expiration:      *resp.Credentials.Expiration,
	}

	if err := u.SaveCredentialsToFile(credentialsFilePath, awsProfile, c); err != nil {
		log.Fatalf("\nError saving credentials to file: %s", err)
	}

	printInfo()
}

func checkProfile() bool {
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

func getCredentialsFilePath() string {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		log.Fatalf("\nError getting home directory: %s", err)
	}

	return filepath.Join(homeDir, ".aws", "credentials")
}

func expirationTimeValid(expirationTime time.Time) bool {
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

func confirmContinuation() bool {
	fmt.Print("🤔 Do you want to continue and generate a new token? (y/n): ")

	// Set the terminal to raw mode to capture a single keystroke
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))

	if err != nil {
		fmt.Printf("\nError setting terminal to raw mode: %s\n", err)
		return false
	}

	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Read a single character
	var response []byte = make([]byte, 1)
	_, err = os.Stdin.Read(response)
	
	if err != nil {
		fmt.Printf("\nError reading response: %s\n", err)
		return false
	}

	// Check if the response is "y"
	return response[0] == 'y'
}

func printInfo() {
	fmt.Printf("\n👍 Credentials saved to file for profile '%s'.\n🎉 You can now use profile '%s' with the AWS CLI.\n", awsProfile, awsProfile)
	color.Yellow("👉 Example: aws configure --profile %s", awsProfile)
	color.Cyan("👌 Example: aws --profile %s s3 ls", awsProfile)
}
