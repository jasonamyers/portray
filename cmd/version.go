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

// configCmd represents the sync command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "display Portray version info",
	Long: `The version command allows you to view information about the current
Portray version`,
	Run: func(cmd *cobra.Command, args []string) {

		// Print the version info
		versionInfo := fmt.Sprintf(`Portray build info:

  Version:	%s
  Git Commit:	%s
  Build Time:	%s
  Go Version:	%s

`, Version, GitCommit, BuildTime, GoVersion)

		fmt.Print(versionInfo)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
