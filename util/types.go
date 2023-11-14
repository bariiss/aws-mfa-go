package util

import "time"

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
