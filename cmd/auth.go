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

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var accountId  string
var userName   string
var tokenCode  string
var profile    string
var noMfa      bool

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "establishes an MFA session via STS",
	Long:  `The auth command helps you authenticate via MFA.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := viper.ReadInConfig() // Find and read the config file
		if accountId == "" && err != nil {
			fmt.Println("No config file available and no account details supplied")
			os.Exit(1)
		}
		if accountId != "" {
			viper.Set("AccountId", accountId)
		}
		if userName != "" {
			viper.Set("UserName", userName)
		}
		usr, err := user.Current()
		checkError(err)
		fileName := usr.HomeDir + "/.aws/portray-session-" + profile + ".json"
		awsCreds := getCredsFromFile(fileName)
		if awsCreds.SessionToken == "" || !validateSession(awsCreds) {
			if tokenCode != "" {
				awsCreds = getNewSession(accountId, userName, tokenCode)
			} else if viper.GetString("AccountId") != "" {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter token: ")
				token, _ := reader.ReadString('\n')
				token = strings.TrimSpace(token)
				awsCreds = getNewSession(
					viper.GetString("AccountId"),
					viper.GetString("UserName"),
					token)
			} else {
				fmt.Println("You need a valid session!")
				os.Exit(1)
			}
			//writeSessionFile(awsCreds, fileName)
		}
		if awsCreds.SessionToken == "" || !validateSession(awsCreds) {
			fmt.Println("You need a valid session!")
			os.Exit(1)
		}

		accountId := awsCreds.AccountId

		sessionToEnvVars(awsCreds, accountId, "", profile)
		startShell(accountId)
	},
}

func init() {
	RootCmd.AddCommand(authCmd)

	authCmd.Flags().StringVarP(&accountId, "account", "a", "", "the AWS account number")
	authCmd.Flags().StringVarP(&userName, "username", "u", "", "the AWS user name")
	authCmd.Flags().StringVarP(&tokenCode, "token", "t", "", "an MFA token")
	authCmd.Flags().StringVarP(&profile, "profile", "p", "default", "a name for your profile")
	authCmd.Flags().BoolP("no-mfa", "n", false, "disable MFA")
    viper.BindPFlag("noMfa", configCmd.Flags().Lookup("no-mfa"))
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func getCredsFromFile(fileName string) (awsCreds AwsCreds) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return
	}

	json.Unmarshal(file, &awsCreds)
	return
}

func validateSession(awsCreds AwsCreds) (valid bool) {
	valid = false
	timestamp := int64(time.Now().Unix())
	if timestamp < awsCreds.Expiration {
		valid = true
	}
	return
}

// AwsCreds represents a set of AWS credentials
type AwsCreds struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      int64
	AccountId       string
}

func getNewSession(accountId string, userName string, tokenCode string) (awsCreds AwsCreds) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	checkError(err)
	svc := sts.New(sess)

	params := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(43200),
		SerialNumber:    aws.String("arn:aws:iam::" + accountId + ":mfa/" + userName),
		TokenCode:       aws.String(tokenCode),
	}

	resp, err := svc.GetSessionToken(params)

	checkError(err)

	awsCreds = AwsCreds{
        *resp.Credentials.AccessKeyId,
		*resp.Credentials.SecretAccessKey,
		*resp.Credentials.SessionToken,
		resp.Credentials.Expiration.Unix(),
		accountId,
	}

	return
}

func sessionToEnvVars(awsCreds AwsCreds, account string, role string, profile string) {
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
	os.Setenv("AWS_SESSION_TOKEN", awsCreds.SessionToken)
	os.Setenv("PORTRAY_PROMPT", prompt)

}

func startShell(account string) {
	fmt.Println("Starting shell with Session in: " + account)
	syscall.Exec(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, syscall.Environ())
}
