package util

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
)

func GenerateMFAToken(secret string) (string, error) {
	token, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", fmt.Errorf("\nError generating TOTP code: %w", err)
	}
	return token, nil
}

func ReadMFADetails(filePath, profile string) (MFADetails, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return MFADetails{}, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("\nError closing file: %s", err)
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
						return MFADetails{}, fmt.Errorf("\nError parsing duration: %w", err)
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
