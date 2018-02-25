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

	"github.com/spf13/cobra"
)

var role string
var profileName string

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "assumes an AWS role",
	Long: `The switch command allows you to assume a role via a named profile
or by passing in the account and role details directly.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("switch called")
	},
}

func init() {
	RootCmd.AddCommand(switchCmd)

	switchCmd.Flags().StringVarP(&accountId, "account", "a", "", "the AWS account number")
	switchCmd.Flags().StringVarP(&role, "role", "r", "", "the AWS role to assume")
	switchCmd.Flags().StringVarP(&profileName, "profile", "p", "", "the name to save these details under")
	switchCmd.Flags().StringVarP(&userName, "username", "u", "", "the AWS user name")
	switchCmd.Flags().BoolP("save", "s", false, "save the account details")
}
