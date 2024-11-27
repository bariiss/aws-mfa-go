package util

import (
	"bufio"
	"fmt"
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
	defer file.Close()

	var details MFADetails
	scanner := bufio.NewScanner(file)

	// Optimize scanner buffer
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Pre-compute profile header
	profileHeader := "[" + profile + "]"
	inProfile := false

	// Pre-allocate commonly used strings
	const (
		deviceKey   = "aws_mfa_device"
		durationKey = "aws_mfa_duration"
		secretKey   = "aws_mfa_secret_key"
	)

	for scanner.Scan() {
		line := scanner.Text()

		// Fast path for empty lines
		if len(line) == 0 {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == profileHeader {
			inProfile = true
			continue
		}

		if inProfile {
			if line[0] == '[' {
				break
			}

			// Only process line if it contains '='
			if idx := strings.IndexByte(line, '='); idx >= 0 {
				key := strings.TrimSpace(line[:idx])
				value := strings.TrimSpace(line[idx+1:])

				switch key {
				case deviceKey:
					details.Device = value
				case durationKey:
					if durationSeconds, err := strconv.ParseInt(value, 10, 64); err == nil {
						details.Duration = time.Duration(durationSeconds * int64(time.Second)).String()
					}
				case secretKey:
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
