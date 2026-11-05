# Deploy role IAM policies

The IAM role GitHub Actions uses to deploy this repo is managed by hand in AWS, not by `template.yaml`. These files are the source of truth for what that role should allow. Change them in a PR first, then apply them.

| File | Applies to |
|------|------------|
| `github-oidc-trust.json` | Trust policy of `financials-api-github-deploy-role`: only `master` of this repo can use the role |
| `github-deploy-policy.json` | Inline policy `financials-api-github-deploy-policy`: deploy the API stack (CloudFormation, Lambda, API Gateway, the stack's IAM role, logs, SAM artifacts) |
| `sam-managed-stack-policy.json` | Inline policy `financials-api-sam-managed-stack-policy`: manage SAM's `aws-sam-cli-managed-default` artifact-bucket stack |
| `cloudformation-template-summary-policy.json` | Inline policy `financials-api-cloudformation-template-summary-policy`: `GetTemplateSummary` for both stacks |

## Placeholders

Fill these in a copy before applying:

| Placeholder | File | Where to find it |
|-------------|------|------------------|
| `<aws-account-id>` | `github-oidc-trust.json` | `aws sts get-caller-identity --query Account --output text` |
| `<api-id>` | `github-deploy-policy.json` | The subdomain of the deployed API URL (`https://<api-id>.execute-api.us-east-1.amazonaws.com`), or the `FinancialsHttpApi` resource in the `financials-api` stack |

The other two policies have no placeholders.

## Applying

Save filled-in copies of the trust policy and the deploy policy in the repo root. `/*-policy*.json` and `/*-trust*.json` are gitignored there, so it can't be committed by accident.

```powershell
$r = "financials-api-github-deploy-role"
aws iam update-assume-role-policy --role-name $r --policy-document file://github-oidc-trust.filled.json
aws iam put-role-policy --role-name $r --policy-name financials-api-github-deploy-policy --policy-document file://github-deploy-policy.filled.json
aws iam put-role-policy --role-name $r --policy-name financials-api-sam-managed-stack-policy --policy-document file://infra/iam/sam-managed-stack-policy.json
aws iam put-role-policy --role-name $r --policy-name financials-api-cloudformation-template-summary-policy --policy-document file://infra/iam/cloudformation-template-summary-policy.json
```

Then run the deploy workflow once to confirm the role still works.

## Notes

- S3 access is limited to SAM's managed artifact bucket (`aws-sam-cli-managed-default-samclisourcebucket-*`).
- API Gateway access is limited to this stack's HTTP API and everything under it, plus listing/creating APIs and tagging. API Gateway permissions are HTTP methods on resource paths, not named actions.
- The API ID is pinned. If CloudFormation ever replaces the API, update `<api-id>` (the UI's `VITE_FINANCIALS_API_BASE_URL` changes too).
- Lambda, IAM and CloudWatch Logs permissions are limited to `financials-api-*` names.
