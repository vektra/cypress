package s3

import (
	"github.com/goamz/goamz/aws"
	"github.com/vektra/cypress"
	"github.com/vektra/errors"
)

type S3Plugin struct {
	Dir       string
	AccessKey string
	SecretKey string
	Bucket    string
	ACL       string
	Region    string
}

func (s *S3Plugin) Receiver() (cypress.Receiver, error) {
	auth := aws.Auth{
		AccessKey: s.AccessKey,
		SecretKey: s.SecretKey,
	}

	acl, ok := validS3ACLs[s.ACL]
	if !ok {
		return nil, errors.Subject(ErrInvalidS3ACL, s.ACL)
	}

	region, ok := aws.Regions[s.Region]
	if !ok {
		region, ok = extraAWSRegions[s.Region]

		if !ok {
			return nil, errors.Subject(ErrInvalidAWSRegion, s.Region)
		}
	}

	return NewS3(s.Dir, s.Bucket, S3Params{ACL: acl, Auth: auth, Region: region})
}

func (s *S3Plugin) Generator() (cypress.Generator, error) {
	auth := aws.Auth{
		AccessKey: s.AccessKey,
		SecretKey: s.SecretKey,
	}

	region, ok := aws.Regions[s.Region]
	if !ok {
		region, ok = extraAWSRegions[s.Region]

		if !ok {
			return nil, errors.Subject(ErrInvalidAWSRegion, s.Region)
		}
	}

	return NewS3Generator(s.Bucket, auth, region)
}

func init() {
	cypress.AddPlugin("S3", func() cypress.Plugin { return &S3Plugin{} })
}
