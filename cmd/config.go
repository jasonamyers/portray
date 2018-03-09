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
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/go-ini/ini"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var sync bool
var outFile string
var format string

type PortrayConfig struct {
	AuthProfiles map[string]AwsAuthProfile `json:auth_profiles`
	Profiles     map[string]AwsRoleProfile `json:profiles`
}

type AwsAuthProfile struct {
	Name      string `json:name`
	AccountId string `json:account_id`
	UserName  string `json:user_name`
	Region    string `json:region`
	Output    string `json:output`
}

type AwsRoleProfile struct {
	Name          string `json:name`
	SourceProfile string `json:source_profile`
	RoleName      string `json:role_name`
	RoleArn       string `json:role_arn`
	MfaSerial     string `json:mfa_serial`
	ExternalId    string `json:external_id`
}

// configCmd represents the sync command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "manage the Portray config",
	Long: `The config command allows you to view the current Portray config,
as well as sync it with the AWS CLI config`,
	Run: func(cmd *cobra.Command, args []string) {
		err := viper.ReadInConfig() // Find and read the config file
		check(err)

		if viper.GetBool("sync") {
			//fmt.Println("Attempting to parse ~/.aws/config")
			parseAwsConfig()
		} else {
			fmt.Println("Bare config command not yet implemented. Try --sync")
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().BoolP("sync", "s", false, "sync Portray config with AWS CLI")
	configCmd.Flags().StringVarP(&outFile, "out-file", "o", "", "The file to save the config to")
	configCmd.Flags().StringVarP(&format, "format", "f", "yaml", "The output format for the config")

	viper.BindPFlag("sync", configCmd.Flags().Lookup("sync"))
	viper.BindPFlag("outFile", configCmd.Flags().Lookup("out-file"))
	viper.BindPFlag("format", configCmd.Flags().Lookup("format"))
}

func parseAwsConfig() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cfg, err := ini.Load(home + "/.aws/config")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	numProfiles := len(cfg.SectionStrings())
	//fmt.Printf("Found %d profiles in AWS config\n", numProfiles)

	portrayConfig := PortrayConfig{}
	awsAuthProfiles := map[string]AwsAuthProfile{}
	awsRoleProfiles := map[string]AwsRoleProfile{}

	awsAuthProfiles = make(map[string]AwsAuthProfile)
	awsRoleProfiles = make(map[string]AwsRoleProfile)

	// loop through all the sections in the ini
	for i := 0; i < numProfiles; i++ {
		sectionHeader := cfg.SectionStrings()[i]
		// skip empty DEFAULT section
		if sectionHeader == "DEFAULT" {
			continue
		} else if sectionHeader == "default" {
			// Found default profile
			//fmt.Printf("Found default profile %s\n", sectionHeader)
		}

		sectionHash := cfg.Section(sectionHeader).KeysHash()
		profileName := strings.Replace(sectionHeader, "profile ", "", 1)

		// Parse out the profiles that don't have a source_profile defined
		// and assume they're a source profile that has credentials.
		if sectionHash["source_profile"] == "" {
			var profile AwsAuthProfile

			profile.Name = profileName
			profile.Region = sectionHash["region"]
			profile.Output = sectionHash["output"]

			awsAuthProfiles[profileName] = profile
		} else {
			var profile AwsRoleProfile

			profile.Name = profileName
			profile.SourceProfile = sectionHash["source_profile"]
			profile.RoleName = strings.Split(sectionHash["role_arn"], "/")[1]
			profile.RoleArn = sectionHash["role_arn"]
			profile.MfaSerial = sectionHash["mfa_serial"]
			profile.ExternalId = sectionHash["external_id"]

			awsRoleProfiles[profileName] = profile
		}
	}

	// For every source profile, let's loop through the role profiles looking
	// for any references to an MFA devices so we can infer account id and
	// username for the source profiles.
	for k, v := range awsAuthProfiles {
		profileName := k
		numReferences := 0

		// See if this profile has any references pointing to it and count them
		for _, values := range awsRoleProfiles {
			if values.SourceProfile == profileName {
				numReferences += 1

				// When we find the first reference, infer account and username
				// details from it and update the AuthProfiles map.
				if numReferences == 1 {
					// grab username from MFA ARN
					userName := strings.Split(values.MfaSerial, "/")[1]
					accountId := strings.Split(values.MfaSerial, ":")[4]

					// Create a temp map to update the fields for the auth
					// profile and copy it back into the awsAuthProfiles map.
					var tmp = v
					tmp.UserName = userName
					tmp.AccountId = accountId
					awsAuthProfiles[profileName] = tmp
				}
			}
		}

		if numReferences > 0 {
			//fmt.Printf("Found %d source references to the %s profile\n", numReferences, profileName)
		}
	}

	portrayConfig.AuthProfiles = awsAuthProfiles
	portrayConfig.Profiles = awsRoleProfiles

	// convert to yaml
	yamlData, err := yaml.Marshal(portrayConfig)
	check(err)
	// convert to json
	jsonData, err := yaml.YAMLToJSON(yamlData)
	check(err)

	// dump yaml to file
	if outFile != "" {
		err = ioutil.WriteFile(outFile, yamlData, 0644)
		check(err)
		fmt.Printf("New configration written to %s\n", outFile)
	} else {
		if format == "yaml" {
			fmt.Println(string(yamlData))
		} else if format == "json" {
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("Unknown output format %s! Valid values are yaml and json\n", format)
			os.Exit(1)
		}
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
