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
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var accountId string
var userName string
var tokenCode string
var profile string
var noMfa bool

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "establishes an MFA session via STS",
	Long:  `The auth command helps you authenticate via MFA.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := viper.ReadInConfig() // Find and read the config file
		checkError(err)

		// If the user doesn't pass in a specific account, try to find it from
		// the default auth profile.
		if accountId == "" {
			defaultAccountId := viper.GetString("AuthProfiles.default.AccountId")
			if defaultAccountId != "" {
				viper.Set("AccountId", defaultAccountId)
			} else {
				fmt.Println("Couldn't find default profile and no account details specified!")
				os.Exit(1)
			}
		} else {
			viper.Set("UserName", userName)
		}

		// If the user doesn't pass in a specific username, try to find it from
		// the default auth profile.
		if userName == "" {
			defaultUserName := viper.GetString("AuthProfiles.default.UserName")
			if defaultUserName != "" {
				viper.Set("UserName", defaultUserName)
			} else {
				fmt.Println("Couldn't find default username!")
				os.Exit(1)
			}
		} else {
			viper.Set("UserName", userName)
		}

		// If the user doesn't pass in a specific profile, try to find it from
		// the default auth profile.
		if profile == "" {
			defaultProfile := viper.GetString("AuthProfiles.default.Name")
			if defaultProfile != "" {
				viper.Set("Profile", defaultProfile)
			} else {
				// Default to zee default
				fmt.Println("Couldn't find default auth profile via config and none specified. Trying \"default\"")
				viper.Set("Profile", "default")
			}
		} else {
			viper.Set("Profile", profile)
		}

		home, err := homedir.Dir()
		checkError(err)
		fileName := home + "/.aws/portray-session-" + viper.GetString("Profile") + ".json"
		awsCreds := getCredsFromFile(fileName)

		// If there's no valid session cache, generate a new session. Prompt
		// for MFA token if it's not passed, unless the --no-mfa flag is set.
		if awsCreds.SessionToken == "" || !validateSession(awsCreds) {
			if tokenCode == "" {
				if viper.GetBool("noMfa") {
					fmt.Println("Skipping MFA token prompting")
				} else {
					// Prompt for MFA token
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("Enter token: ")
					token, _ := reader.ReadString('\n')
					tokenCode = strings.TrimSpace(token)
				}
			}

			awsCreds = getNewSession(
				viper.GetString("AccountId"),
				viper.GetString("UserName"),
				tokenCode)

			writeSessionFile(awsCreds, fileName)
		} else {
			// Found a cached sessions that's still valid
			fmt.Println("Using cached session credentials")

			// Check how much time is left on the session so we can tell the user
			sessionExpiration := time.Unix(awsCreds.Expiration, 0)
			currentTime := time.Now()
			sessionTimeLeft := sessionExpiration.Sub(currentTime)

			fmt.Printf("Session valid for %+v\n", Round(sessionTimeLeft, time.Second))
		}

		sessionToEnvVars(
			awsCreds,
			viper.GetString("AccountId"),
			"",
			viper.GetString("Profile"))

		startShell(viper.GetString("AccountId"))
	},
}

func init() {
	rootCmd.AddCommand(authCmd)

	authCmd.Flags().StringVarP(&accountId, "account", "a", "", "the AWS account number")
	authCmd.Flags().StringVarP(&userName, "username", "u", "", "the AWS user name")
	authCmd.Flags().StringVarP(&tokenCode, "token", "t", "", "an MFA token")
	authCmd.Flags().StringVarP(&profile, "profile", "p", "", "a name for your profile")
	authCmd.Flags().BoolP("no-mfa", "n", false, "disable MFA")

	viper.BindPFlag("noMfa", authCmd.Flags().Lookup("no-mfa"))
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

func writeSessionFile(awsCreds AwsCreds, fileName string) {
	awsCredsJSON, _ := json.Marshal(awsCreds)

	createFile(fileName)
	err := ioutil.WriteFile(fileName, awsCredsJSON, 0600)
	checkError(err)
}

func createFile(path string) {
	// detect if file exists
	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		checkError(err)
		defer file.Close()
	}
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
