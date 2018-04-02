// Copyright © 2017 Jason Myers <jason@mailthemyers.com>
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

// AwsCreds represents a set of AWS credentials
type AwsCreds struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      int64
	AccountId       string
}

func WriteSessionFile(awsCreds AwsCreds, fileName string) {
	awsCredsJSON, _ := json.Marshal(awsCreds)

	createFile(fileName)
	err := ioutil.WriteFile(fileName, awsCredsJSON, 0600)
	CheckError(err)
}

func createFile(path string) {
	// detect if file exists
	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		CheckError(err)
		defer file.Close()
	}
}

func Round(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}

func CheckError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func GetCredsFromFile(fileName string) (awsCreds AwsCreds) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return
	}

	json.Unmarshal(file, &awsCreds)
	return
}

func ValidateSession(awsCreds AwsCreds) (valid bool) {
	valid = false
	timestamp := int64(time.Now().Unix())
	if timestamp < awsCreds.Expiration {
		valid = true
	}
	return
}

func GetNewSession(profile string, accountId string, userName string, tokenCode string) (awsCreds AwsCreds) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config:  aws.Config{Region: aws.String("us-east-1")},
		Profile: profile,
	}))
	svc := sts.New(sess)

	// If no tokenCode is passed, assume MFA has been disabled by a flag
	var params *sts.GetSessionTokenInput
	if tokenCode == "" {
		params = &sts.GetSessionTokenInput{
			DurationSeconds: aws.Int64(43200),
		}
	} else {
		params = &sts.GetSessionTokenInput{
			DurationSeconds: aws.Int64(43200),
			SerialNumber:    aws.String("arn:aws:iam::" + accountId + ":mfa/" + userName),
			TokenCode:       aws.String(tokenCode),
		}
	}

	resp, err := svc.GetSessionToken(params)
	CheckError(err)

	awsCreds = AwsCreds{
		*resp.Credentials.AccessKeyId,
		*resp.Credentials.SecretAccessKey,
		*resp.Credentials.SessionToken,
		resp.Credentials.Expiration.Unix(),
		accountId,
	}

	return
}

func GetNewRoleSession(accountId string, roleName string, externalId string, usr user.User) (awsCreds AwsCreds) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	CheckError(err)
	svc := sts.New(sess)

	timestamp := int64(time.Now().Unix())
	if externalId == "" {
		externalId = usr.Username
	}

	params := &sts.AssumeRoleInput{
		ExternalId:      aws.String(externalId),
		DurationSeconds: aws.Int64(3600),
		RoleArn:         aws.String("arn:aws:iam::" + accountId + ":role/" + roleName),
		RoleSessionName: aws.String("Portray-" + usr.Username + "-" + strconv.FormatInt(timestamp, 10)),
	}

	resp, err := svc.AssumeRole(params)
	CheckError(err)

	awsCreds = AwsCreds{
		*resp.Credentials.AccessKeyId,
		*resp.Credentials.SecretAccessKey,
		*resp.Credentials.SessionToken,
		resp.Credentials.Expiration.Unix(),
		accountId,
	}

	return
}

func SessionToEnvVars(awsCreds AwsCreds, account string, role string, profile string) {
	prompt := account
	if role != "" {
		prompt = prompt + ":" + role
	}
	if profile != "" {
		prompt = prompt + ":" + profile
	}

	fmt.Println("Setting ENV VARS")
	os.Setenv("AWS_ACCESS_KEY_ID", awsCreds.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", awsCreds.SecretAccessKey)
	os.Setenv("AWS_SECURITY_TOKEN", awsCreds.SessionToken)
	os.Setenv("AWS_SESSION_TOKEN", awsCreds.SessionToken)
	os.Setenv("PORTRAY_PROMPT", prompt)
}

func StartShell(sessionName string) {
	fmt.Println("Starting shell with Session in: " + sessionName)
	syscall.Exec(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, syscall.Environ())
}
