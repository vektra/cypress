package s3

import (
	"os"

	"github.com/goamz/goamz/aws"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
	"github.com/vektra/errors"
)

type Send struct {
	Dir       string `short:"d" long:"dir" description:"directory to use for intermediate data"`
	AccessKey string `short:"a" long:"access" description:"AWS access key"`
	SecretKey string `short:"s" long:"secret" description:"AWS secret key"`
	Bucket    string `short:"b" long:"bucket" description:"bucket to store data in"`

	ACL    string `long:"acl" description:"ACL to apply to data"`
	Region string `short:"r" long:"region" description:"AWS region to use"`
}

func (s *Send) Execute(args []string) error {
	auth := aws.Auth{
		AccessKey: s.AccessKey,
		SecretKey: s.SecretKey,
	}

	acl, ok := validS3ACLs[s.ACL]
	if !ok {
		return errors.Subject(ErrInvalidS3ACL, s.ACL)
	}

	region, ok := aws.Regions[s.Region]
	if !ok {
		region, ok = extraAWSRegions[s.Region]

		if !ok {
			return errors.Subject(ErrInvalidAWSRegion, s.Region)
		}
	}

	r, err := NewS3(s.Dir, s.Bucket, S3Params{ACL: acl, Auth: auth, Region: region})
	if err != nil {
		return err
	}

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(dec, r)
}

type Recv struct {
	Dir       string `short:"d" long:"dir" description:"directory to use for intermediate data"`
	AccessKey string `short:"a" long:"access" description:"AWS access key"`
	SecretKey string `short:"s" long:"secret" description:"AWS secret key"`
	Bucket    string `short:"b" long:"bucket" description:"bucket to store data in"`

	Region string `short:"r" long:"region" description:"AWS region to use"`
}

func (s *Recv) Execute(args []string) error {
	auth := aws.Auth{
		AccessKey: s.AccessKey,
		SecretKey: s.SecretKey,
	}

	region, ok := aws.Regions[s.Region]
	if !ok {
		region, ok = extraAWSRegions[s.Region]

		if !ok {
			return errors.Subject(ErrInvalidAWSRegion, s.Region)
		}
	}

	enc := cypress.NewStreamEncoder(os.Stdout)
	gen, err := NewS3Generator(s.Bucket, auth, region)
	if err != nil {
		return err
	}

	return cypress.Glue(gen, enc)
}

func init() {
	commands.Add("s3:send", "write message streams to S3", "", &Send{})
	commands.Add("s3:recv", "read messages back from S3", "", &Recv{})
}
