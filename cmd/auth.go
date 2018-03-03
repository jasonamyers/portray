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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jasonamyers/portray/util"
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
		noMfa = viper.GetBool("NoMfa")

		// User specified profile
		if profile != "" {
			// validate it against Portray config
			if viper.IsSet("AuthProfiles." + profile) {
				profileKey := "AuthProfiles." + profile + "."
				fmt.Printf("Found profile %s in config\n", profile)

				// get account id from profile
				if accountId == "" {
					if !viper.IsSet(profileKey + "AccountId") {
						fmt.Printf("Error! Unable to find AccountId for the %s profile. Is it configured in the AuthProfiles section?\n", profile)
						os.Exit(1)
					}
					accountId = viper.GetString(profileKey + "AccountId")
					viper.Set("AccountId", accountId)
				} else {
					fmt.Println("Error! Can't specify alternate account for a configured profile")
					os.Exit(1)
				}

				// get user name from profile
				if userName == "" {
					if !viper.IsSet(profileKey + "UserName") {
						fmt.Printf("Error! Unable to find UserName for the %s profile. Is it configured in the AuthProfiles section?\n", profile)
						os.Exit(1)
					}
					userName = viper.GetString(profileKey + "UserName")
					viper.Set("UserName", userName)
				} else {
					fmt.Println("Error! Can't specify alternate username for a configured profile")
					os.Exit(1)
				}

				// passed validations, tell dah user
				fmt.Printf("Using %s profile with AccountId %s and UserName %s\n",
					profile,
					accountId,
					userName)

			} else {
				fmt.Printf("Invalid profile %s! Is it configured in the AuthProfiles section?\n", profile)
				os.Exit(1)
			}
			// user has not specified a profile
		} else {
			// user has not specified account
			// try to find default from config
			if accountId == "" {
				defaultAccountId := viper.GetString("AuthProfiles.default.AccountId")
				defaultUserName := viper.GetString("AuthProfiles.default.UserName")
				defaultProfileName := viper.GetString("AuthProfiles.default.Name")

				// populate account id
				if defaultAccountId != "" {
					accountId = defaultAccountId
					viper.Set("AccountId", defaultAccountId)
				} else {
					fmt.Println("Error! Unable to find AccoundId for the default profile. Is it configured in the AuthProfiles section?")
				}

				// populate user name
				if defaultUserName != "" {
					userName = defaultUserName
					viper.Set("UserName", defaultUserName)
				} else {
					fmt.Println("Error! Unable to find UserName for the default profile. Is it configured in the AuthProfiles section?")
				}

				// populate profile name
				if defaultProfileName != "" {
					profile = defaultProfileName
					viper.Set("Profile", defaultProfileName)
				} else {
					fmt.Println("Default profile name not specified in config. Using default AWS profile \"default\"")
					profile = "default"
					viper.Set("Profile", profile)
				}

				fmt.Printf("Using default profile %s with AccountId %s and UserName %s\n", profile, accountId, userName)
			} else {
				// if user has specified account, they have to specify username
				// as well.
				if userName == "" {
					fmt.Println("Error! Must specify --username/-u if manually setting AccountId via --account/-a")
					os.Exit(1)
				} else {
					viper.Set("UserName", userName)
				}
			}
		}

		home, err := homedir.Dir()
		util.CheckError(err)
		fileName := home + "/.aws/portray-session-" + profile + ".json"
		awsCreds := util.GetCredsFromFile(fileName)

		// If there's no valid session cache, generate a new session. Prompt
		// for MFA token if it's not passed, unless the --no-mfa flag is set.
		if awsCreds.SessionToken == "" || !util.ValidateSession(awsCreds) {
			if tokenCode == "" {
				if noMfa {
					fmt.Println("Skipping MFA token prompting")
				} else {
					// Prompt for MFA token
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("Enter token: ")
					token, _ := reader.ReadString('\n')
					tokenCode = strings.TrimSpace(token)
				}
			}

			awsCreds = util.GetNewSession(profile, accountId, userName, tokenCode)
			util.WriteSessionFile(awsCreds, fileName)
		} else {
			// Found a cached sessions that's still valid
			fmt.Println("Using cached session credentials")

			// Check how much time is left on the session so we can tell the user
			sessionExpiration := time.Unix(awsCreds.Expiration, 0)
			currentTime := time.Now()
			sessionTimeLeft := sessionExpiration.Sub(currentTime)

			fmt.Printf("Session valid for %+v\n", util.Round(sessionTimeLeft, time.Second))
		}

		util.SessionToEnvVars(awsCreds, accountId, "", profile)
		util.StartShell(accountId)
	},
}

func init() {
	rootCmd.AddCommand(authCmd)

	authCmd.Flags().StringVarP(&accountId, "account", "a", "", "the AWS account number")
	authCmd.Flags().StringVarP(&userName, "username", "u", "", "the AWS user name")
	authCmd.Flags().StringVarP(&tokenCode, "token", "t", "", "an MFA token")
	authCmd.Flags().StringVarP(&profile, "profile", "p", "", "a name for your profile")
	authCmd.Flags().BoolP("no-mfa", "n", false, "disable MFA")

	viper.BindPFlag("AccountId", authCmd.Flags().Lookup("account"))
	viper.BindPFlag("UserName", authCmd.Flags().Lookup("username"))
	viper.BindPFlag("Profile", authCmd.Flags().Lookup("profile"))
	viper.BindPFlag("NoMfa", authCmd.Flags().Lookup("no-mfa"))
}
