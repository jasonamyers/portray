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
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/jasonamyers/portray/util"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var roleAccountId string
var roleArn string
var roleName string
var roleExternalId string
var roleProfile string

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Assumes an AWS role",
	Long: `The switch command allows you to assume a role via a named profile
or by passing in the account and role details directly.`,
	Run: func(cmd *cobra.Command, args []string) {
		if roleProfile == "" && (roleAccountId == "" || roleName == "") {
			fmt.Println("Error! Use either a named profile or manually specify both account and role")
			fmt.Println("See portray switch -h for options")
			os.Exit(1)
		}

		if roleProfile != "" {
			if viper.IsSet("Profiles." + roleProfile) {
				profileKey := "Profiles." + roleProfile + "."
				fmt.Printf("Found profile %s in config\n", roleProfile)

				if roleAccountId != "" {
					fmt.Println("Error! Can't specify alternate account for a configured profile")
					os.Exit(1)
				}

				if roleName != "" {
					fmt.Println("Error! Can't specify alternate role name for a configured profile")
					os.Exit(1)
				}

				// get role arn from profile
				roleArn = viper.GetString(profileKey + "RoleArn")
				if roleArn == "" {
					fmt.Println("Error! Couldn't find RoleArn in profile config")
					os.Exit(1)
				}
				// get role name from role arn
				roleName = strings.Split(roleArn, "/")[1]
				// get account id from role arn
				roleAccountId = strings.Split(roleArn, ":")[4]
				// get external id from profile
				roleExternalId = viper.GetString(profileKey + "ExternalId")

			} else {
				fmt.Printf("Error! Unable to find profile %s in config. Is it set in the Profiles section?\n", roleProfile)
				os.Exit(1)
			}
		} else { // user has not specified profile
			// user has not specified account
			if roleAccountId == "" || roleName == "" {
				fmt.Println("Error! When not using named profiles, you must specify both the account and the role name")
				os.Exit(1)
			}
		}

		currentUser, err := user.Current()
		util.CheckError(err)

		home, err := homedir.Dir()
		util.CheckError(err)
		roleFileName := home + "/.aws/portray-role-session-" + roleAccountId + "_" + roleName + ".json"
		awsCreds := util.GetCredsFromFile(roleFileName)

		// If there's no valid session cache, generate a new session.
		if awsCreds.SessionToken == "" || !util.ValidateSession(awsCreds) {
			fmt.Printf("No session cache found or cache expired. Assuming role %s in account %s\n", roleName, roleAccountId)

			awsCreds = util.GetNewRoleSession(
				roleAccountId,
				roleName,
				roleExternalId,
				*currentUser)

			util.WriteSessionFile(awsCreds, roleFileName)
		} else {
			// Found a cached sessions that's still valid
			fmt.Println("Using cached session credentials")

			// Check how much time is left on the session so we can tell the user
			sessionExpiration := time.Unix(awsCreds.Expiration, 0)
			currentTime := time.Now()
			sessionTimeLeft := sessionExpiration.Sub(currentTime)

			fmt.Printf("Session valid for %+v\n", util.Round(sessionTimeLeft, time.Second))
		}

		util.SessionToEnvVars(awsCreds, roleAccountId, roleName, roleProfile)
		util.StartShell(roleAccountId)
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)

	switchCmd.Flags().StringVarP(&roleAccountId, "account", "a", "", "the 12-digit AWS account ID")
	switchCmd.Flags().StringVarP(&roleName, "role", "r", "", "the name of the role to assume")
	switchCmd.Flags().StringVarP(&roleExternalId, "external-id", "e", "", "the ExternalId required to assume the role")
	switchCmd.Flags().StringVarP(&roleProfile, "profile", "p", "", "the named profile to use (conflicts w/ others)")

	viper.BindPFlag("AccountId", switchCmd.Flags().Lookup("account"))
	viper.BindPFlag("Role", switchCmd.Flags().Lookup("role"))
	viper.BindPFlag("ExternalId", switchCmd.Flags().Lookup("external-id"))
	viper.BindPFlag("Profile", switchCmd.Flags().Lookup("profile"))
}
