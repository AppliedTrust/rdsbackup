#rdsbackup

######Use **rdsbackup** to copy your RDS snapshots to another AWS region for backup/DR purposes.

####Example: `rdsbackup --source=us-east-1 --dest=us-west-1 db_name`

####Usage:
```
  rdsbackup [options] <db_instance_id>
  rdsbackup -h --help
  rdsbackup --version

AWS Authentication:
  Either use the -K and -S flags, or
  set the AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables.

Options:
  -s, --source=<region>     AWS region of source RDS instance [default: us-east-1].
  -d, --dest=<region>       AWS region to store backup RDS snapshot [default: us-west-1].
  -p, --purge=<count>       Purge oldest snapshots from dest region if more than <count> exist.
  -q, --quiet               Silence all output except errors.
  -h, --help                Show this screen.
  --version                 Show version.
```

####Purging old snapshots:
AWS imposes a limit of 50 RDS snapshots per region. Use the **--purge** flag to purge an instance's oldest snapshots from the destination region. **rdsbackup** will never purge snapshots from the source region.

**rdsbackup** tags all snapshots it creates as follows.  It will only purge snapshots in the destination region that that have a "managedby" tag set to "rdsbackup". 

tag | value
--- | -----
managedby | rdsbackup
source | ``<SourceRegion>``
sourceid | ``<SourceDBIdentifier>``
sourcearn | ``<SourceDBARN>``
time | ``<2006-01-02 15:04:05 -0700>``
timestamp | ``<SecondsSinceEpoch>``

####Configuring AWS credentials:
See the README for the AWS SDK for more details: https://github.com/aws/aws-sdk-go#configuring-credentials
