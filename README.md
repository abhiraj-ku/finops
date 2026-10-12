# FinOps

Small Go CLI for finding idle AWS EBS volumes and RDS instances.

It:

- finds unattached EBS volumes and available RDS instances;
- checks the last 14 days of CloudWatch activity;
- estimates monthly cost;
- looks up the creator in the last 90 days of CloudTrail;
- writes a `cleanup.sh` remediation script.

The scan is read-only. The generated script creates snapshots and deletes
resources, so review it before running.

## Design

```text
EC2/RDS -> CloudWatch idle check -> CloudTrail owner lookup -> report + cleanup.sh
```

The CLI runs locally and calls AWS directly. There is no server, database,
container, or Terraform configuration.

Cost estimates are approximate:

- EBS: `$0.08/GB-month`, or `$0.125/GB-month` for `io1`/`io2`
- RDS: class-based compute estimate plus `$0.115/GB-month` storage

## Requirements

- Go 1.25.5+
- AWS credentials configured through the AWS SDK default credential chain
- AWS CLI, only if running `cleanup.sh`

The scan role needs:

```text
ec2:DescribeVolumes
rds:DescribeDBInstances
cloudwatch:GetMetricData
cloudtrail:LookupEvents
```

The cleanup role additionally needs:

```text
ec2:CreateSnapshot
ec2:DeleteVolume
rds:DeleteDBInstance
```

## Setup

```bash
git clone https://github.com/abhiraj-ku/finops.git
cd finops
go mod download
aws configure
aws sts get-caller-identity
```

Do not commit credentials. The repository does not load `.env` files.

## Run

```bash
# Default: ap-south-1
go run ./cmd/finops

# Choose a region
go run ./cmd/finops -region ap-south-1
```

If candidates are found:

```bash
less cleanup.sh
./cleanup.sh
```

`cleanup.sh` has no confirmation prompt. Run it only after reviewing the
resources and using an identity with the cleanup permissions.

## Development

```bash
gofmt -w cmd internal
go vet ./...
```

## Limits

- Only EBS volumes and RDS instances are supported.
- Missing CloudWatch data can produce false positives.
- Creator lookup is limited to 90 days.
- Cost figures are estimates, not AWS billing data.
