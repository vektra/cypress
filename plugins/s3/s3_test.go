package s3

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/goamz/goamz/s3/s3test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/keystore"
	"github.com/vektra/cypress/plugins/spool"
	"github.com/vektra/neko"
)

func TestS3(t *testing.T) {
	n := neko.Start(t)

	var (
		s3s       *s3test.Server
		s3c       *s3.S3
		s3a       *S3
		awsAuth   aws.Auth
		awsRegion aws.Region
	)

	cypress.EmptyGlobalConfig = true

	bucketName := "test-logs"

	tmpdir, err := ioutil.TempDir("", "s3-cypress")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	spooldir := filepath.Join(tmpdir, "spool")

	s3cfg := cypress.GlobalConfig().S3
	defKeys := keystore.Default()

	n.Setup(func() {
		var err error
		s3s, err = s3test.NewServer(nil)
		require.NoError(t, err)

		awsRegion = aws.Region{
			Name:                 "faux-region-1",
			S3LocationConstraint: true,
			S3Endpoint:           s3s.URL(),
		}

		awsAuth = aws.Auth{AccessKey: "abc", SecretKey: "123"}
		s3c = s3.New(awsAuth, awsRegion)

		s3a, err = NewS3(spooldir, bucketName, s3.Private, awsAuth, awsRegion)
		require.NoError(t, err)

		err = s3c.Bucket(bucketName).PutBucket(s3.Private)
		require.NoError(t, err)
	})

	n.Cleanup(func() {
		cypress.GlobalConfig().S3 = s3cfg
		keystore.SetDefault(defKeys)

		os.RemoveAll(spooldir)
		s3s.Quit()
	})

	n.It("saves messages to a disk buffer that is sent to S3 when flushed", func() {
		m := cypress.Log()
		m.Add("hello", "world")

		err := s3a.Receive(m)
		require.NoError(t, err)

		fileData, err := ioutil.ReadFile(s3a.CurrentFile())
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		bucket := s3c.Bucket(bucketName)

		data, err := bucket.Get(s3a.LastFile())
		require.NoError(t, err)

		assert.Equal(t, string(fileData), string(data))
	})

	n.It("can interface directly with an existing spooler", func() {
		spooldir2 := filepath.Join(tmpdir, "spool2")
		defer os.RemoveAll(spooldir2)

		spool, err := spool.NewSpool(spooldir2)
		require.NoError(t, err)

		m := cypress.Log()
		m.Add("hello", "world")

		s, err := NewS3WithSpool(spool, bucketName, s3.Private, awsAuth, awsRegion)
		require.NoError(t, err)

		err = spool.Receive(m)
		require.NoError(t, err)

		fileData, err := ioutil.ReadFile(spool.CurrentFile())
		require.NoError(t, err)

		err = spool.Rotate()
		require.NoError(t, err)

		bucket := s3c.Bucket(bucketName)

		data, err := bucket.Get(s.LastFile())
		require.NoError(t, err)

		assert.Equal(t, string(fileData), string(data))
	})

	n.It("can generate logs reading from an s3 bucket", func() {
		m := cypress.Log()
		m.Add("hello", "world")

		err := s3a.Receive(m)
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		gen, err := NewS3Generator(bucketName, awsAuth, awsRegion)
		require.NoError(t, err)

		gen.AllowUnsigned = true

		m2, err := gen.Generate()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("deals with there being no logs", func() {
		gen, err := NewS3Generator(bucketName, awsAuth, awsRegion)
		require.NoError(t, err)

		_, err = gen.Generate()
		require.Equal(t, err, io.EOF)
	})

	n.It("can sign the data when it's uploaded", func() {
		var tk keystore.TestKeys

		s3a.SignWith(tk.Gen())

		m := cypress.Log()
		m.Add("hello", "world")

		err := s3a.Receive(m)
		require.NoError(t, err)

		fileData, err := ioutil.ReadFile(s3a.CurrentFile())
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		bucket := s3c.Bucket(bucketName)

		resp, err := bucket.GetResponse(s3a.LastFile())
		require.NoError(t, err)

		var gen S3Generator
		gen.Keys = &tk

		signature, err := gen.extractSignature(resp)
		require.NoError(t, err)

		data, err := bucket.Get(s3a.LastFile())
		require.NoError(t, err)

		assert.True(t, signature.KeyID != "")

		assert.NoError(t, signature.ValidateETag(resp))

		assert.Equal(t, string(fileData), string(data))
	})

	n.It("automatically uses a key specified in the global config", func() {
		cypress.GlobalConfig().S3.SignKey = "foo"

		tk := &keystore.TestKeys{
			Name: "foo",
		}

		keystore.SetDefault(tk)

		key := tk.Gen()

		s3a, err = NewS3(spooldir, bucketName, s3.Private, awsAuth, awsRegion)
		require.NoError(t, err)

		assert.Equal(t, key, s3a.signKey)
	})

	n.It("reads logs from s3 files in time order", func() {
		m := cypress.Log()
		m.Add("source", "old")

		err := s3a.Receive(m)
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		cm := cypress.Log()
		cm.Add("source", "current")

		err = s3a.Receive(cm)
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		gen, err := NewS3Generator(bucketName, awsAuth, awsRegion)
		require.NoError(t, err)

		gen.AllowUnsigned = true

		assert.Equal(t, 2, len(gen.List().Contents))

		m2, err := gen.Generate()
		require.NoError(t, err)

		assert.Equal(t, m, m2)

		m3, err := gen.Generate()
		require.NoError(t, err)

		assert.Equal(t, cm, m3)
	})

	n.It("reads logs from s3 files in multiple batches", func() {
		m := cypress.Log()
		m.Add("source", "old")

		err := s3a.Receive(m)
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		cm := cypress.Log()
		cm.Add("source", "current")

		err = s3a.Receive(cm)
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		gen, err := NewS3Generator(bucketName, awsAuth, awsRegion)
		require.NoError(t, err)

		gen.AllowUnsigned = true

		gen.listMax = 1
		gen.list = nil
		gen.marker = ""

		err = gen.updateList()
		require.NoError(t, err)

		assert.Equal(t, 1, len(gen.List().Contents))

		m2, err := gen.Generate()
		require.NoError(t, err)

		assert.Equal(t, m, m2)

		m3, err := gen.Generate()
		require.NoError(t, err)

		assert.Equal(t, cm, m3)
	})

	n.It("can verify logs when read back", func() {
		var tk keystore.TestKeys

		s3a.SignWith(tk.Gen())

		m := cypress.Log()
		m.Add("hello", "world")

		err := s3a.Receive(m)
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		gen, err := NewS3Generator(bucketName, awsAuth, awsRegion)
		require.NoError(t, err)

		gen.Keys = &tk

		m2, err := gen.Generate()
		require.NoError(t, err)

		assert.Equal(t, m, m2)

		signature := gen.LastSignature()
		assert.True(t, signature.KeyID != "")
	})

	n.It("returns an error if the logs have no signature", func() {
		var tk keystore.TestKeys

		s3a.SignWith(tk.Gen())

		m := cypress.Log()
		m.Add("hello", "world")

		err := s3a.Receive(m)
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		name := s3a.LastFile()

		var buf bytes.Buffer

		enc := cypress.NewStreamEncoder(&buf)

		err = enc.Init(cypress.SNAPPY)
		require.NoError(t, err)

		m.Add("host", "foobar")

		err = enc.Receive(m)
		require.NoError(t, err)

		err = s3c.Bucket(bucketName).Put(name, buf.Bytes(), "application/binary", s3.Private, s3.Options{})
		require.NoError(t, err)

		gen, err := NewS3Generator(bucketName, awsAuth, awsRegion)
		require.NoError(t, err)

		gen.Keys = &tk

		_, err = gen.Generate()
		require.Error(t, err)
	})

	n.It("returns an error if the signature doesn't validate", func() {
		var tk keystore.TestKeys

		s3a.SignWith(tk.Gen())

		m := cypress.Log()
		m.Add("hello", "world")

		err := s3a.Receive(m)
		require.NoError(t, err)

		err = s3a.Rotate()
		require.NoError(t, err)

		name := s3a.LastFile()

		var buf bytes.Buffer

		enc := cypress.NewStreamEncoder(&buf)

		err = enc.Init(cypress.SNAPPY)
		require.NoError(t, err)

		m.Add("host", "foobar")

		err = enc.Receive(m)
		require.NoError(t, err)

		resp, err := s3c.Bucket(bucketName).GetResponse(name)
		require.NoError(t, err)

		options := s3.Options{
			Meta: map[string][]string{
				SignatureHeaderKey: []string{resp.Header.Get(SignatureHeader)},
			},
		}

		err = s3c.Bucket(bucketName).Put(name, buf.Bytes(), "application/binary", s3.Private, options)
		require.NoError(t, err)

		gen, err := NewS3Generator(bucketName, awsAuth, awsRegion)
		require.NoError(t, err)

		gen.Keys = &tk

		_, err = gen.Generate()
		require.Error(t, err)
	})

	n.Meow()
}

func TestS3Plugin(t *testing.T) {
	n := neko.Start(t)

	tmpdir, err := ioutil.TempDir("", "log")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	var (
		s3s       *s3test.Server
		s3c       *s3.S3
		awsAuth   aws.Auth
		awsRegion aws.Region
	)

	cypress.EmptyGlobalConfig = true

	bucketName := "test-logs"

	n.Setup(func() {
		var err error
		s3s, err = s3test.NewServer(nil)
		require.NoError(t, err)

		awsRegion = aws.Region{
			Name:                 "faux-region-1",
			S3LocationConstraint: true,
			S3Endpoint:           s3s.URL(),
		}

		extraAWSRegions[awsRegion.Name] = awsRegion

		awsAuth = aws.Auth{AccessKey: "abc", SecretKey: "123"}
		s3c = s3.New(awsAuth, awsRegion)

		err = s3c.Bucket(bucketName).PutBucket(s3.Private)
		require.NoError(t, err)
	})

	n.It("can create an S3 receiver", func() {
		plug := &S3Plugin{
			Dir:       tmpdir,
			AccessKey: "foo",
			SecretKey: "bar",
			Bucket:    "xxyyzz",
			ACL:       "public",
			Region:    "us-west-1",
		}

		_, err := plug.Receiver()
		require.NoError(t, err)
	})

	n.It("errors out if the acl is unknown", func() {
		plug := &S3Plugin{
			ACL: "blah",
		}

		_, err := plug.Receiver()
		require.Error(t, err)
	})

	n.It("errors out if the region is unknown", func() {
		plug := &S3Plugin{
			ACL:    "public",
			Region: "mars-1",
		}

		_, err := plug.Receiver()
		require.Error(t, err)
	})

	n.It("can create an S3 generator", func() {
		plug := &S3Plugin{
			Dir:       tmpdir,
			AccessKey: "foo",
			SecretKey: "bar",
			Bucket:    bucketName,
			ACL:       "public",
			Region:    awsRegion.Name,
		}

		_, err := plug.Generator()
		require.NoError(t, err)
	})

	n.Meow()
}
