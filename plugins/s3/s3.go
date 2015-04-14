package s3

import (
	"crypto/ecdsa"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"os"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/httputil"
	"github.com/vektra/cypress/keystore"
	"github.com/vektra/cypress/plugins/spool"
	"github.com/vektra/errors"
	"github.com/vektra/tai64n"
)

type S3Plugin struct {
	Dir       string
	AccessKey string
	SecretKey string
	Bucket    string
	ACL       string
	Region    string
}

var validS3ACLs = map[string]s3.ACL{
	"private":                   s3.Private,
	"public-read":               s3.PublicRead,
	"public-read-write":         s3.PublicReadWrite,
	"authenticated-read":        s3.AuthenticatedRead,
	"bucket-owner-read":         s3.BucketOwnerRead,
	"bucket-owner-full-control": s3.BucketOwnerFull,

	// Some more easier aliases
	"public":      s3.PublicRead,
	"0644":        s3.PublicRead,
	"world-write": s3.PublicReadWrite,
	"0666":        s3.PublicReadWrite,
	"auth-read":   s3.AuthenticatedRead,
	"0040":        s3.AuthenticatedRead,
	"owner-read":  s3.BucketOwnerRead,
	"0400":        s3.BucketOwnerRead,
	"owner":       s3.BucketOwnerFull,
	"0600":        s3.BucketOwnerFull,
}

var extraAWSRegions = map[string]aws.Region{}

var (
	ErrInvalidS3ACL     = errors.New("invalid s3 ACL")
	ErrInvalidAWSRegion = errors.New("invalid AWS region")
)

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

type S3Config struct {
	SignKey       string
	AllowUnsigned bool
}

type S3 struct {
	ACL s3.ACL

	client   *s3.S3
	bucket   *s3.Bucket
	spool    *spool.Spool
	enc      *cypress.StreamEncoder
	lastFile string

	signKey *ecdsa.PrivateKey
}

type S3Params struct {
	Config *cypress.Config
	ACL    s3.ACL
	Auth   aws.Auth
	Region aws.Region
}

func (p *S3Params) Client() *s3.S3 {
	return s3.New(p.Auth, p.Region)
}

func (p *S3Params) S3Config() *S3Config {
	cfg := p.Config
	if cfg == nil {
		cfg = cypress.GlobalConfig()
	}

	var s3cfg S3Config

	cfg.Load("s3", &s3cfg)

	return &s3cfg
}

func NewS3(dir, bucket string, params S3Params) (*S3, error) {
	spool, err := spool.NewSpool(dir)
	if err != nil {
		return nil, err
	}

	client := params.Client()

	s3 := &S3{
		ACL:    params.ACL,
		client: client,
		bucket: client.Bucket(bucket),
		spool:  spool,
	}

	err = s3.setupKey(params)
	if err != nil {
		return nil, err
	}

	spool.OnRotate = s3.onRotate

	return s3, nil
}

func NewS3WithSpool(spool *spool.Spool, bucket string, params S3Params) (*S3, error) {
	client := params.Client()

	s3 := &S3{
		ACL:    params.ACL,
		client: client,
		bucket: client.Bucket(bucket),
		spool:  spool,
	}

	err := s3.setupKey(params)
	if err != nil {
		return nil, err
	}

	spool.OnRotate = s3.onRotate

	return s3, nil
}

func (s *S3) setupKey(params S3Params) error {
	key, err := params.SignKey()
	if err != nil {
		return err
	}

	s.signKey = key

	return nil
}

func (p *S3Params) SignKey() (*ecdsa.PrivateKey, error) {
	cfg := p.S3Config()

	if cfg.SignKey == "" {
		return nil, nil
	}

	return keystore.Default().GetPrivate(cfg.SignKey)
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
	ErrMissingSignature = errors.New("missing signature")
	ErrMissingETag      = errors.New("missing ETag to verify")
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
		return ErrMissingETag
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

	// Indicates if we should process any unsigned logs seen
	AllowUnsigned bool

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

	var cfg S3Config

	err := cypress.GlobalConfig().Load("s3", &cfg)
	if err != nil {
		return nil, err
	}

	gen := &S3Generator{
		AllowUnsigned: cfg.AllowUnsigned,
		client:        client,
		bucket:        client.Bucket(bucket),
		cur:           -1,
		listMax:       100,
	}

	err = gen.updateList()
	if err != nil {
		return nil, err
	}

	return gen, nil
}

func (gen *S3Generator) extractSignature(resp *http.Response) (*S3Signature, error) {
	var sig S3Signature

	data := resp.Header.Get(SignatureHeader)
	if data == "" {
		if gen.AllowUnsigned {
			return &sig, nil
		}

		return nil, ErrMissingSignature
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

	if len(list.Contents) > 0 {
		g.marker = list.Contents[len(list.Contents)-1].Key
	}

	return nil
}

func (g *S3Generator) List() *s3.ListResp {
	return g.list
}

func (g *S3Generator) LastSignature() *S3Signature {
	return g.signature
}

// This is used because objects can now have a GLACIER
// class and we want to ignore it. Rather than looking
// for GLACIER explicitly, we look for the ones we want
// because amazon might add new ones.
func (g *S3Generator) validClass(class string) bool {
	switch class {
	case "", "STANDARD", "REDUCED_REDUNDANCY":
		return true
	default:
		return false
	}
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

		if !g.validClass(g.list.Contents[g.cur].StorageClass) {
			goto restart
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

func (g *S3Generator) Close() error {
	g.dec.Close()

	if g.response != nil && g.response.Body != nil {
		g.response.Body.Close()
	}

	return nil
}
