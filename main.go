package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	u "github.com/bariiss/aws-mfa-go/util"
	"github.com/fatih/color"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var (
	awsProfile, awsRegion = os.Getenv("AWS_MFA_GO_USER"), os.Getenv("AWS_MFA_GO_REGION")
)

func main() {
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

	expirationTime, err := u.ReadExpirationTime(credentialsFilePath, awsProfile)
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

	mfaDetails, err := u.ReadMFADetails(credentialsFilePath, awsProfile+"-go")
	if err != nil {
		fmt.Printf("Error reading MFA details: %s\n", err)
		return
	}

	stsClient := sts.NewFromConfig(cfg)

	mfaToken, err := u.GenerateMFAToken(mfaDetails.SecretKey)
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

	// save credentials to file as profile without -go suffix
	err = u.SaveCredentialsToFile(credentialsFilePath, awsProfile, c)
	if err != nil {
		log.Fatalf("Error saving credentials to file: %s", err)
	}

	printInfo()
}

func checkProfile() bool {
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

		return true
	}
	return false
}

func printInfo() {
	fmt.Printf("👍 Credentials saved to file for profile '%s'.\n", awsProfile)
	fmt.Printf("🎉 You can now use profile '%s' with the AWS CLI.\n", awsProfile)
	fmt.Println("👉 Please check your AWS CLI config file to make sure you are using the correct profile name.")
	info := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("👉 Example: %s\n", info("aws configure --profile "+awsProfile))

	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println("👉 You can also use the profile with the AWS CLI directly.")
	fmt.Printf("👌 Example: %s\n", cyan("aws --profile "+awsProfile+" s3 ls"))
}
