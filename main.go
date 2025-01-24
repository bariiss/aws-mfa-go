package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	u "github.com/bariiss/aws-mfa-go/util"
	"github.com/fatih/color"
)

var (
	awsProfile = os.Getenv("AWS_MFA_GO_USER")
	awsRegion  = os.Getenv("AWS_MFA_GO_REGION")
)

func main() {
	if u.CheckProfile(awsProfile, awsRegion) {
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(awsProfile+"-go"),
		config.WithRegion(awsRegion),
	)

	if err != nil {
		log.Fatalf("\nUnable to load AWS config: %s", err)
	}

	credentialsFilePath := u.GetCredentialsFilePath()
	expirationTime, err := u.ReadExpirationTime(credentialsFilePath, awsProfile)

	if err != nil {
		log.Printf("\nError reading expiration time: %s", err)
	}

	if u.ExpirationTimeValid(awsProfile, expirationTime) {
		if !u.ConfirmContinuation() {
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

	fmt.Printf("\n👍 Credentials saved to file for profile '%s'.\n🎉 You can now use profile '%s' with the AWS CLI.\n", awsProfile, awsProfile)
	color.Yellow("👉 Example: aws configure --profile %s", awsProfile)
	color.Cyan("👌 Example: aws --profile %s s3 ls", awsProfile)
}
