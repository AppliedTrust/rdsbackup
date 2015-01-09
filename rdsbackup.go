package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/stripe/aws-go/aws"
	"github.com/stripe/aws-go/gen/iam"
	"github.com/stripe/aws-go/gen/rds"
	"log"
	"os"
	"strings"
	"time"
)

const version = "1.0"

var usage = `rdsbackup: easy cross-region AWS RDS backups

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
`

type config struct {
	dbid      string
	src       string
	dst       string
	arn       string
	copyId    string
	awsKeyId  string
	awsSecret string
	debugOn   bool
	creds     aws.CredentialsProvider
}

func main() {
	c, err := parseArgs()
	if err != nil {
		log.Fatal(err)
	}
	// TODO: cleanup old snapshots
	err = c.findLatestSnap()
	if err != nil {
		log.Fatal(err)
	}
	err = c.copySnap()
	if err != nil {
		log.Fatal(err)
	}
	err = c.waitForCopy()
	if err != nil {
		log.Fatal(err)
	}
	c.debug("All done!")
}

// parseArgs
func parseArgs() (config, error) {
	c := config{}
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		return c, err
	}
	c.dbid = args["<db_instance_id>"].(string)
	c.src = args["--source"].(string)
	c.dst = args["--dest"].(string)
	c.debugOn = args["--debug"].(bool)
	if arg, ok := args["--awskey"].(string); ok {
		c.awsKeyId = arg
	} else {
		c.awsKeyId = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if arg, ok := args["--awssecret"].(string); ok {
		c.awsSecret = arg
	} else {
		c.awsSecret = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}
	if len(c.awsKeyId) < 1 || len(c.awsSecret) < 1 {
		return c, fmt.Errorf("Must use -K and -S options or set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables.")
	}
	c.creds = aws.Creds(c.awsKeyId, c.awsSecret, "")
	return c, nil
}

// findAcccountID
func (c *config) findAcccountID() (string, error) {
	i := iam.New(c.creds, c.src, nil)
	u, err := i.GetUser(nil)
	if err != nil {
		return "", err
	}
	parts := strings.Split(*u.User.ARN, ":")
	if len(parts) != 6 {
		return "", fmt.Errorf("Error parsing user ARN")
	}
	return parts[4], nil
}

// findLatestSnap
func (c *config) findLatestSnap() error {
	awsacctid, err := c.findAcccountID()
	if err != nil {
		return err
	}
	cli := rds.New(c.creds, c.src, nil)
	c.debug(fmt.Sprintf("Searching for snapshots for: %s\n", c.dbid))
	q := rds.DescribeDBSnapshotsMessage{}
	q.DBInstanceIdentifier = aws.String(c.dbid)
	resp, err := cli.DescribeDBSnapshots(&q)
	if err != nil {
		return err
	}
	newest := time.Unix(0, 0)
	newestId := ""
	if len(resp.DBSnapshots) < 1 {
		return fmt.Errorf("No snapshots found")
	}
	c.debug(fmt.Sprintf("Found %d snapshots for: %s\n", len(resp.DBSnapshots), c.dbid))
	for _, r := range resp.DBSnapshots {
		if r.SnapshotCreateTime.After(newest) {
			newestId = *r.DBSnapshotIdentifier
			newest = r.SnapshotCreateTime
		}
	}
	if len(newestId) < 1 {
		return fmt.Errorf("No usable snapshot found")
	}
	c.arn = fmt.Sprintf("arn:aws:rds:%s:%s:snapshot:%s", c.src, awsacctid, newestId)
	c.debug(fmt.Sprintf("Copying latest snapshot: %s: %s \n", newestId, newest.String()))
	return nil
}

// copySnap
func (c *config) copySnap() error {
	cli := rds.New(c.creds, c.dst, nil)
	t := time.Now()
	c.copyId = fmt.Sprintf("%s-%s", c.dbid, t.Format("2006-01-02at15-04MST"))
	m := rds.CopyDBSnapshotMessage{
		SourceDBSnapshotIdentifier: aws.String(c.arn),
		Tags: []rds.Tag{
			rds.Tag{aws.String("time"), aws.String(t.Format("2006-01-02 15:04:05 -0700"))},
			rds.Tag{aws.String("timestamp"), aws.String(fmt.Sprintf("%d", t.Unix()))},
			rds.Tag{aws.String("source"), aws.String(c.src)},
			rds.Tag{aws.String("sourceid"), aws.String(c.dbid)},
			rds.Tag{aws.String("managedby"), aws.String("rdsbackup")},
		},
		TargetDBSnapshotIdentifier: aws.String(c.copyId),
	}
	resp, err := cli.CopyDBSnapshot(&m)
	if err != nil {
		return err
	} else if *resp.DBSnapshot.Status != "creating" {
		return fmt.Errorf("Error creating snapshot - unexpected status: %s", *resp.DBSnapshot.Status)
	}
	return nil
}

// waitForCopy
func (c *config) waitForCopy() error {
	c.debug(fmt.Sprintf("Waiting for copy %s...\n", c.copyId))
	cli := rds.New(c.creds, c.dst, nil)
	q := rds.DescribeDBSnapshotsMessage{}
	q.DBSnapshotIdentifier = aws.String(c.copyId)
	for {
		resp, err := cli.DescribeDBSnapshots(&q)
		if err != nil {
			return err
		}
		if len(resp.DBSnapshots) != 1 {
			return fmt.Errorf("New snapshot missing!")
		}
		s := resp.DBSnapshots[0]
		if *s.Status != "creating" {
			break
		}
		c.debug(fmt.Sprintf("Waiting %s (%d%% complete)", *s.Status, *s.PercentProgress))
		time.Sleep(10 * time.Second) // Run for some time to simulate work
	}
	return nil
}

// debug
func (c *config) debug(s string) {
	if c.debugOn {
		log.Println(s)
	}
}
