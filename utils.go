package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/api/crypt"
)

var (
	// bucket, key, no region
	regexpS3UrlStyle1 = regexp.MustCompile(`^https?://([^.]+).s3.amazonaws.com(/?$|/(.*))`)
	// bucket, region, key
	regexpS3UrlStyle2 = regexp.MustCompile(`^https?://([^.]+).s3-([^.]+).amazonaws.com(/?$|/(.*))`)
	// bucket, key, no region
	regexpS3UrlStyle3 = regexp.MustCompile(`^https?://s3.amazonaws.com/([^\/]+)(/?$|/(.*))`)
	// region, bucket, key
	regexpS3UrlStyle4 = regexp.MustCompile(`^https?://s3-([^.]+).amazonaws.com/([^\/]+)(/?$|/(.*))`)
)

// ParseS3Url - Parse all styles of the s3 buckets so requests
// can be made through the api.
//
// For Reference: http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingBucket.html
//
// returns bucket, key, region
func ParseS3Url(url string) (string, string, string, error) {
	matches := regexpS3UrlStyle1.FindStringSubmatch(url)

	if len(matches) > 0 {
		return matches[1], matches[3], "", nil
	}

	matches = regexpS3UrlStyle2.FindStringSubmatch(url)
	if len(matches) > 0 {
		return matches[1], matches[4], matches[2], nil
	}

	matches = regexpS3UrlStyle3.FindStringSubmatch(url)
	if len(matches) > 0 {
		return matches[1], matches[3], "us-east-1", nil
	}

	matches = regexpS3UrlStyle4.FindStringSubmatch(url)
	if len(matches) > 0 {
		return matches[2], matches[4], matches[1], nil
	}

	return "", "", "", errors.New("not an s3 url")
}

// NewCipher - Creates a new Crypt object (a cipher) for encryption/decryption
func NewCipher() (*crypt.Crypt, error) {
	sess, err := session.NewSession()

	if err != nil {
		return nil, err
	}

	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		return nil, err
	}

	return &crypt.Crypt{
		AwsRegion: *sess.Config.Region,
		AwsToken:  creds.SessionToken,
		AwsAccess: creds.AccessKeyID,
		AwsSecret: creds.SecretAccessKey,
	}, nil
}

func s3Svc() (*s3.S3, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return s3.New(sess), nil
}

func s3GetObject(url string) ([]byte, error) {
	s3Bucket, s3Key, _, err := ParseS3Url(url)
	if err != nil {
		return nil, err
	}

	svc, err := s3Svc()
	if err != nil {
		return nil, err
	}
	input := s3.GetObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3Key),
	}
	resp, err := svc.GetObject(&input)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func s3PutObject(url string, data []byte) error {
	s3Bucket, s3Key, _, err := ParseS3Url(url)
	if err != nil {
		return err
	}

	svc, err := s3Svc()
	if err != nil {
		return err
	}

	input := s3.PutObjectInput{
		Bucket:      aws.String(s3Bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	}

	_, err = svc.PutObject(&input)
	return err
}

// Escapes single quotes for bash
func escapeSingleQuote(s string) string {
	return strings.Replace(s, "'", "'\"'\"'", -1)
}
