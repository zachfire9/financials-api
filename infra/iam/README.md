# Deploy role IAM policies

The IAM role GitHub Actions uses to deploy this repo is managed by hand in AWS, not by `template.yaml`. These files are the source of truth for what that role should allow. Change them in a PR first, then apply them.

| File | Applies to |
|------|------------|
| `github-oidc-trust.json` | Trust policy of `financials-api-github-deploy-role`: only `master` of this repo can use the role |
| `github-deploy-policy.json` | Inline policy `financials-api-github-deploy-policy`: deploy the API stack (CloudFormation, Lambda, API Gateway, the stack's IAM role, logs, SAM artifacts) |
| `sam-managed-stack-policy.json` | Inline policy `financials-api-sam-managed-stack-policy`: manage SAM's `aws-sam-cli-managed-default` artifact-bucket stack |
| `cloudformation-template-summary-policy.json` | Inline policy `financials-api-cloudformation-template-summary-policy`: `GetTemplateSummary` for both stacks |

## Placeholders

Only the trust policy has one: replace `<aws-account-id>` (`aws sts get-caller-identity --query Account --output text`). The permission policies use wildcards for the region and account.

## Applying

Save a filled-in copy of the trust policy in the repo root. `/*-policy*.json` and `/*-trust*.json` are gitignored there, so it can't be committed by accident.

```powershell
$r = "financials-api-github-deploy-role"
aws iam update-assume-role-policy --role-name $r --policy-document file://github-oidc-trust.filled.json
aws iam put-role-policy --role-name $r --policy-name financials-api-github-deploy-policy --policy-document file://infra/iam/github-deploy-policy.json
aws iam put-role-policy --role-name $r --policy-name financials-api-sam-managed-stack-policy --policy-document file://infra/iam/sam-managed-stack-policy.json
aws iam put-role-policy --role-name $r --policy-name financials-api-cloudformation-template-summary-policy --policy-document file://infra/iam/cloudformation-template-summary-policy.json
```

Then run the deploy workflow once to confirm the role still works.

## Known gaps

Two statements in `github-deploy-policy.json` are broader than they need to be:

- `SamArtifactBuckets` allows reading and writing objects in **any** S3 bucket. It only needs SAM's managed artifact bucket (`aws-sam-cli-managed-default-samclisourcebucket-*`).
- `ApiGatewayDeployFinancialsApi` allows **all** API Gateway actions on all APIs. It only needs the API created by this stack.
