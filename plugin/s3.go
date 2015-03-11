package plugin

import (
	"crypto/ecdsa"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"os"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/httputil"
	"github.com/vektra/cypress/keystore"
	"github.com/vektra/tai64n"
)

type S3 struct {
	ACL  s3.ACL
	Keys keystore.Keys

	client   *s3.S3
	bucket   *s3.Bucket
	spool    *Spool
	enc      *cypress.StreamEncoder
	lastFile string

	signKey *ecdsa.PrivateKey
}

func NewS3(dir, bucket string, acl s3.ACL, auth aws.Auth, region aws.Region) (*S3, error) {
	spool, err := NewSpool(dir)
	if err != nil {
		return nil, err
	}

	client := s3.New(auth, region)

	s3 := &S3{
		ACL:    acl,
		Keys:   keystore.Default(),
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

func (s *S3) SignWith(k *ecdsa.PrivateKey) {
	s.signKey = k
}

func (s *S3) onRotate(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}

	mh := md5.New()

	size, err := io.Copy(mh, f)
	if err != nil {
		return err
	}

	sum := mh.Sum(nil)

	acl := s.ACL
	opts := s3.Options{
		ContentMD5: base64.StdEncoding.EncodeToString(sum),
	}

	_, err = f.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}

	if s.signKey != nil {
		sigr, sigs, err := ecdsa.Sign(rand.Reader, s.signKey, sum)
		if err != nil {
			return err
		}

		sig := &S3Signature{KeyID: keystore.KeyId(&s.signKey.PublicKey), R: sigr, S: sigs}

		jsdata, err := json.Marshal(sig)
		if err != nil {
			return err
		}

		opts.Meta = map[string][]string{
			SignatureHeaderKey: []string{
				base64.StdEncoding.EncodeToString(jsdata),
			},
		}
	}

	fileName := tai64n.Now().Label()

	s.lastFile = fileName

	return s.bucket.PutReader(fileName, f, size, httputil.BinaryLogContentType, acl, opts)
}

var (
	ErrCorruptSignature = errors.New("corrupt signature")
	ErrInvalidSignature = errors.New("invalid signature")
)

const (
	SignatureHeaderKey = "cypress-signature"
	SignatureHeader    = "x-amz-meta-" + SignatureHeaderKey
)

type S3Signature struct {
	Keys  keystore.Keys `json:"-"`
	KeyID string        `json:"key_id"`
	R     *big.Int      `json:"r"`
	S     *big.Int      `json:"s"`
}

func (sig *S3Signature) ValidateETag(resp *http.Response) error {
	if sig.KeyID == "" {
		return nil
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		return ErrInvalidSignature
	}

	sum, err := hex.DecodeString(etag)
	if err != nil {
		return err
	}

	pkey, err := sig.Keys.Get(sig.KeyID)
	if err != nil {
		return err
	}

	if !ecdsa.Verify(pkey, sum, sig.R, sig.S) {
		return ErrInvalidSignature
	}

	return nil
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
	Keys keystore.Keys

	client *s3.S3
	bucket *s3.Bucket

	list    *s3.ListResp
	files   []string
	marker  string
	listMax int

	cur int
	dec *cypress.StreamDecoder

	response  *http.Response
	signature *S3Signature
}

func NewS3Generator(bucket string, auth aws.Auth, region aws.Region) (*S3Generator, error) {
	client := s3.New(auth, region)

	gen := &S3Generator{
		client:  client,
		bucket:  client.Bucket(bucket),
		cur:     -1,
		listMax: 100,
	}

	err := gen.updateList()
	if err != nil {
		return nil, err
	}

	return gen, nil
}

func (gen *S3Generator) extractSignature(resp *http.Response) (*S3Signature, error) {
	var sig S3Signature

	data := resp.Header.Get(SignatureHeader)
	if data == "" {
		return &sig, nil
	}

	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, ErrCorruptSignature
	}

	err = json.Unmarshal(bytes, &sig)
	if err != nil {
		return nil, ErrCorruptSignature
	}

	sig.Keys = gen.Keys

	return &sig, nil
}

func (g *S3Generator) updateList() error {
	if g.list != nil && !g.list.IsTruncated {
		g.list = nil
		return nil
	}

	list, err := g.bucket.List("@", "", g.marker, g.listMax)
	if err != nil {
		return err
	}

	g.list = list
	g.marker = list.Contents[len(list.Contents)-1].Key

	return nil
}

func (g *S3Generator) List() *s3.ListResp {
	return g.list
}

func (g *S3Generator) LastSignature() *S3Signature {
	return g.signature
}

func (g *S3Generator) Generate() (*cypress.Message, error) {
restart:

	if g.dec == nil {
		g.cur++

		if g.cur == len(g.list.Contents) {
			err := g.updateList()
			if err != nil {
				return nil, err
			}

			if g.list == nil {
				return nil, io.EOF
			}

			g.cur = 0
		}

		resp, err := g.bucket.GetResponse(g.list.Contents[g.cur].Key)
		if err != nil {
			return nil, err
		}

		signature, err := g.extractSignature(resp)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}

		err = signature.ValidateETag(resp)
		if err != nil {
			return nil, err
		}

		g.signature = signature

		dec, err := cypress.NewStreamDecoder(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}

		g.response = resp
		g.dec = dec
	}

	m, err := g.dec.Generate()
	if err == io.EOF {
		g.response.Body.Close()
		g.dec = nil
		goto restart
	}

	return m, err
}
