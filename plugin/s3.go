package plugin

import (
	"crypto/md5"
	"encoding/base64"
	"io"
	"os"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/httputil"
	"github.com/vektra/tai64n"
)

type S3 struct {
	ACL      s3.ACL
	client   *s3.S3
	bucket   *s3.Bucket
	spool    *Spool
	enc      *cypress.StreamEncoder
	lastFile string
}

func NewS3(dir, bucket string, acl s3.ACL, auth aws.Auth, region aws.Region) (*S3, error) {
	spool, err := NewSpool(dir)
	if err != nil {
		return nil, err
	}

	client := s3.New(auth, region)

	s3 := &S3{
		ACL:    acl,
		client: client,
		bucket: client.Bucket(bucket),
		spool:  spool,
	}

	spool.OnRotate = s3.onRotate

	return s3, nil
}

func NewS3WithSpool(spool *Spool, bucket string, acl s3.ACL, auth aws.Auth, region aws.Region) (*S3, error) {
	client := s3.New(auth, region)

	s3 := &S3{
		ACL:    acl,
		client: client,
		bucket: client.Bucket(bucket),
		spool:  spool,
	}

	spool.OnRotate = s3.onRotate

	return s3, nil
}

func (s *S3) onRotate(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}

	h := md5.New()

	size, err := io.Copy(h, f)
	if err != nil {
		return err
	}

	acl := s.ACL
	opts := s3.Options{
		ContentMD5: base64.StdEncoding.EncodeToString(h.Sum(nil)),
	}

	_, err = f.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}

	fileName := tai64n.Now().Label()

	s.lastFile = fileName

	return s.bucket.PutReader(fileName, f, size, httputil.BinaryLogContentType, acl, opts)
}

func (s3 *S3) Receive(m *cypress.Message) error {
	return s3.spool.Receive(m)
}

func (s *S3) CurrentFile() string {
	return s.spool.CurrentFile()
}

func (s3 *S3) LastFile() string {
	return s3.lastFile
}

func (s3 *S3) Rotate() error {
	return s3.spool.Rotate()
}

type S3Generator struct {
	client *s3.S3
	bucket *s3.Bucket

	files []string
	cur   int
	dec   *cypress.StreamDecoder
}

func NewS3Generator(bucket string, auth aws.Auth, region aws.Region) (*S3Generator, error) {
	client := s3.New(auth, region)

	gen := &S3Generator{
		client: client,
		bucket: client.Bucket(bucket),
		cur:    -1,
	}

	err := gen.getList()
	if err != nil {
		return nil, err
	}

	return gen, nil
}

func (g *S3Generator) getList() error {
	list, err := g.bucket.List("@", "", "", 1000)
	if err != nil {
		return err
	}

	for _, key := range list.Contents {
		g.files = append(g.files, key.Key)
	}

	return nil
}

func (g *S3Generator) Files() []string {
	return g.files
}

func (g *S3Generator) Generate() (*cypress.Message, error) {
restart:

	if g.dec == nil {
		g.cur++

		if g.cur == len(g.files) {
			return nil, io.EOF
		}

		r, err := g.bucket.GetReader(g.files[g.cur])
		if err != nil {
			return nil, err
		}

		dec, err := cypress.NewStreamDecoder(r)
		if err != nil {
			return nil, err
		}

		g.dec = dec
	}

	m, err := g.dec.Generate()
	if err == io.EOF {
		g.dec = nil
		goto restart
	}

	return m, err
}
