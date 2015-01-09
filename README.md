##rdsbackup: easy cross-region AWS RDS backups

```
rdsbackup: easy cross-region AWS RDS backups

Usage:
  rdsbackup [options] <db_instance_id>
  rdsbackup -h --help
  rdsbackup --version

AWS Authentication:
  Either use the -K and -S flags, or
  set the AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables.

Options:
  -s, --source=<region>     AWS region of source RDS instance [default: us-east-1].
  -d, --dest=<region>       AWS region to store backup RDS snapshot [default: us-west-1].
  -K, --awskey=<keyid>      AWS key ID (or use AWS_ACCESS_KEY_ID environemnt variable).
  -S, --awssecret=<secret>  AWS secret key (or use AWS_SECRET_ACCESS_KEY environemnt variable).
  --debug                   Enable debugging output.
  --version                 Show version.
  -h, --help                Show this screen.
```
