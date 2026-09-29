# GitHub Actions API deploy

This repo deploys the Financials API from GitHub Actions when changes are merged into `master`. The workflow uses GitHub OIDC to assume an AWS IAM role, so no long-lived AWS access keys should be stored in GitHub.

Workflow file: `.github/workflows/deploy-api.yml`

## GitHub configuration

Add these repository settings in GitHub before enabling the workflow.

### Secrets

- `AWS_DEPLOY_ROLE_ARN`: ARN of the AWS IAM role that GitHub Actions can assume for this repo.
- `FINANCIALS_ACCESS_TOKEN`: private shared token passed to the SAM stack as `FinancialsAccessToken`.

### Variables

- `AWS_REGION`: AWS region for the API stack, for example `us-east-1`.
- `API_STACK_NAME`: CloudFormation stack name, for example `financials-api`.
- `FINANCIALS_STORAGE_DRIVER`: normally `ephemeral` for browser-owned sessions.
- `FINANCIALS_ALLOWED_ORIGINS`: comma-separated static UI origins allowed by CORS, for example `https://<custom-domain>`.

Keep real deployed values in GitHub settings or private operator notes, not committed config files.

## AWS OIDC role setup

Use a trust policy scoped to this repository and `master` branch. Replace placeholders before applying:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::<aws-account-id>:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
          "token.actions.githubusercontent.com:sub": "repo:zachfire9/financials-api:ref:refs/heads/master"
        }
      }
    }
  ]
}
```

The role needs enough permissions to run `sam deploy` for the existing API stack and read the deployed stack output for the health smoke test. Scope the policy to this stack/resources where practical:

- CloudFormation describe/create/update changesets and stacks for the API stack
- Lambda update/create permissions for the API function
- API Gateway HTTP API permissions used by the SAM stack
- IAM role create/update/pass permissions for the Lambda execution role managed by the stack
- S3 read/write permissions for the SAM deployment artifact bucket
- CloudWatch Logs permissions if the stack manages log groups

## Workflow behavior

On `push` to `master`, and on manual `workflow_dispatch`, the workflow runs three separate jobs:

1. `Test API`
   - Checks out the repo.
   - Sets up Go 1.22.
   - Runs `go test ./...`.
2. `Build API artifact`
   - Runs after tests pass.
   - Checks out the repo and sets up Go 1.22.
   - Installs the AWS SAM CLI.
   - Runs `sam build`.
   - Uploads `.aws-sam/build/` as the `financials-api-sam-build` artifact.
3. `Deploy API stack`
   - Runs after the SAM build succeeds.
   - Downloads the `financials-api-sam-build` artifact.
   - Installs the AWS SAM CLI.
   - Assumes the AWS deploy role through OIDC.
   - Runs `sam deploy --template-file .aws-sam/build/template.yaml --resolve-s3` with GitHub variables/secrets as parameter overrides. `--resolve-s3` lets SAM create/use its managed artifact bucket in the target account instead of requiring a committed bucket name.
   - Reads the API URL from CloudFormation outputs and calls `/health`.

## Rollback

To roll back an API deploy, either:

- revert the bad commit on `master` and let the workflow redeploy, or
- manually deploy a known-good checkout with the same private parameters:

```powershell
sam build; sam deploy --stack-name financials-api --profile zachfire9 --capabilities CAPABILITY_IAM --parameter-overrides "FinancialsStorageDriver=ephemeral" "FinancialsAllowedOrigins=<allowed-origins>" "FinancialsAccessToken=<private-access-token>"
```
