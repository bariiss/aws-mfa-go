# AWS MFA Go

A robust Go application for managing AWS Multi-Factor Authentication (MFA) session tokens. This tool simplifies the process of obtaining and managing temporary credentials for AWS CLI and SDK operations when MFA is enabled.

## Features

- Automated MFA token generation and validation
- Session token management with configurable duration
- Secure credential file handling
- Support for multiple AWS profiles
- Cross-platform support (Linux, macOS, Windows)
- Easy integration with existing AWS workflows

## Prerequisites

- Go 1.21 or later
- AWS CLI configured with base credentials
- AWS IAM user with MFA enabled
- AWS credentials file (`~/.aws/credentials`)

## Installation

### Using Go Install

```bash
go install github.com/bariiss/aws-mfa-go@latest
```

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/bariiss/aws-mfa-go.git
cd aws-mfa-go
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build -o aws-mfa-go
```

## Configuration

### Environment Variables

Set the following environment variables:

```bash
export AWS_MFA_GO_USER=your-aws-profile
export AWS_MFA_GO_REGION=your-aws-region
```

Add these to your shell configuration file (`.bashrc`, `.zshrc`, etc.) for persistence.

### AWS Credentials

1. Ensure your base AWS credentials are configured in `~/.aws/credentials`:

```ini
[your-aws-profile]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
aws_mfa_device = arn:aws:iam::ACCOUNT_ID:mfa/YOUR_USERNAME
aws_mfa_duration = 43200  # Optional: Session duration in seconds (default: 12 hours)
aws_mfa_secret_key = YOUR_MFA_SECRET  # For virtual MFA devices
```

## Usage

1. Run the application:
```bash
aws-mfa-go
```

2. The tool will:
   - Generate MFA token automatically (if secret key is configured)
   - Request MFA code (if manual input is needed)
   - Obtain temporary credentials from AWS
   - Save the credentials to your AWS credentials file

3. Use AWS CLI or SDK with the new profile:
```bash
aws s3 ls --profile your-aws-profile-go
```

## Security Considerations

- Store MFA secret keys securely
- Use appropriate session durations
- Keep your AWS credentials file protected
- Regularly rotate access keys
- Never commit credentials to version control

## Building for Different Platforms

The application can be built for various platforms:

```bash
# For Linux
GOOS=linux GOARCH=amd64 go build -o aws-mfa-go-linux

# For macOS
GOOS=darwin GOARCH=amd64 go build -o aws-mfa-go-macos

# For Windows
GOOS=windows GOARCH=amd64 go build -o aws-mfa-go.exe
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- AWS SDK for Go
- The Go community
- All contributors to this project