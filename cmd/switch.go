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
				//profileKey := "Profiles." + roleProfile + "."
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
				// get external id from profile
			} else {
				fmt.Printf("Error! Unable to find profile %s in config. Is it set in the Profiles section?\n", roleProfile)
			}
		}
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
