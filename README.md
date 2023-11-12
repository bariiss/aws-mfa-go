
# AWS MFA Go

AWS MFA Go is a Go application designed to manage session tokens associated with MFA (Multi-Factor Authentication) for AWS (Amazon Web Services).

## Features

- Manages AWS credentials and automates the login process with MFA.
- Generates temporary credentials for AWS CLI and SDK.

## Installation

Follow these steps to run the project:

1. Clone or download the project.
2. Install the required Go modules:

   ```bash
   go mod download
   ```

3. Run the application:

   ```bash
   go run main.go
   ```

4. Install the application: (optional)

   ```bash
   go install github.com/bariiss/aws-mfa-go@latest
   aws-mfa-go # NOTE: You must set the environment variables and credentials before running the application.
   ```

## Usage

Set the following environment variables to start the application:

- `AWS_MFA_GO_USER`: Your AWS username (associated with the MFA device).
- `AWS_MFA_GO_REGION`: Your AWS region.

You can also set the environment variables in the `~/.bashrc` file:

```bash
export AWS_MFA_GO_USER=<profile>
export AWS_MFA_GO_REGION=<region>
```

Save your AWS credentials in the `~/.aws/credentials` file:

```bash
[<profile>-go] # The profile name must end with "-go".
aws_access_key_id = <aws_access_key_id> # Your AWS access key ID.
aws_secret_access_key = <aws_secret_access_key> # Your AWS secret access key.
aws_mfa_device = <aws_mfa_device> # Your AWS MFA device ARN.
aws_mfa_duration = <aws_mfa_duration> # The duration of the session token (in seconds).
aws_mfa_secret_key = <aws_mfa_secret_key> # Your AWS MFA secret key.
```

You can create a new profile by running the `aws-mfa-go` command:

```bash
aws-mfa-go
```

After setting the environment variables and saving your AWS credentials, you can run the `aws configure` command to configure your AWS CLI:

```bash
aws configure --profile <profile>
```

Then, you can manage your AWS MFA operations by running the application.

## Contributing

You can contribute by sending pull requests or opening issues for bug reports and feature requests.