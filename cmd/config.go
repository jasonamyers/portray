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
    "strconv"
	"strings"

    "github.com/ghodss/yaml"
	"github.com/go-ini/ini"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type PortrayConfig struct {
    DefaultProfile   string           `json:default_profile`
    DefaultAccountId int              `json:default_account_id`
    AuthProfiles     []AwsAuthProfile `json:auth_profiles`
    Profiles         []AwsRoleProfile `json:profiles`
//    AuthProfiles map[string]map[string]AwsAuthProfile `json:source_profiles`
//    Profiles       map[string]map[string]AwsRoleProfile   `json:profiles`
}

type AwsAuthProfile struct {
	Name      string `json:name`
    AccountId int    `json:account_id`
    UserName  string `json:user_name`
	Region    string `json:region`
	Output    string `json:output`
}

type AwsRoleProfile struct {
	Name          string `json:name`
	SourceProfile string `json:source_profile`
	RoleName      string `json:role_name`
	MfaSerial     string `json:mfa_serial`
	ExternalId    string `json:external_id`
}

// configCmd represents the sync command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "manage the Portray config",
	Long:  `The config command allows you to view the current Portray config,
as well as sync it with the AWS CLI config`,
	Run: func(cmd *cobra.Command, args []string) {
		err := viper.ReadInConfig() // Find and read the config file
        check(err)

        if viper.GetBool("sync") {
            fmt.Println("Attempting to parse ~/.aws/config")
		    parseAwsConfig()
        } else {
            fmt.Println("Bare config command not yet implemented. Try --sync")
        }
	},
}

func init() {
	RootCmd.AddCommand(configCmd)

	configCmd.Flags().BoolP("sync", "s", false, "sync Portray config with AWS CLI")
    viper.BindPFlag("sync", configCmd.Flags().Lookup("sync"))
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

    // ini parsing has a DEFAULT section that's empty. Let's not count it.
	numProfiles := len(cfg.SectionStrings()) - 1
	fmt.Printf("Found %d profiles in AWS config\n", numProfiles)

    portrayConfig   := PortrayConfig{}
    awsAuthProfiles := []AwsAuthProfile{}
    awsRoleProfiles := []AwsRoleProfile{}

	for i := 0; i < numProfiles; i++ {
        sectionHeader := cfg.SectionStrings()[i]
        // skip empty DEFAULT section
        if sectionHeader == "DEFAULT" {
            continue
        } else if sectionHeader == "default" {
            // Found default profile
            portrayConfig.DefaultProfile = "default"
        }

        sectionHash := cfg.Section(sectionHeader).KeysHash()
        profileName := strings.Replace(sectionHeader, "profile ", "", 1)

        // Parse out the profiles that don't have a source_profile defined
        // and assume they're a source profile that has credentials.
        if sectionHash["source_profile"] == "" {
            var profile AwsAuthProfile

            profile.Name      = profileName
            profile.Region    = sectionHash["region"] 
            profile.Output    = sectionHash["output"] 
            awsAuthProfiles = append(awsAuthProfiles, profile)
        } else {
            var profile AwsRoleProfile

            profile.Name          = profileName
	        profile.SourceProfile = sectionHash["source_profile"]
	        profile.RoleName      = strings.Split(sectionHash["role_arn"], "/")[1]
	        profile.MfaSerial     = sectionHash["mfa_serial"]
	        profile.ExternalId    = sectionHash["external_id"]
            awsRoleProfiles       = append(awsRoleProfiles, profile)
        }
	}

    numAuthProfiles := len(awsAuthProfiles)
    numRoleProfiles   := len(awsRoleProfiles)
    // For every source profile, let's loop through the role profiles looking
    // for any references to an MFA devices so we can infer account id and
    // username for the source profiles.
	for i := 0; i < numAuthProfiles; i++ {
        profileName := awsAuthProfiles[i].Name
            
        for x := 0; x < numRoleProfiles; x++ {
            sourceProfile := awsRoleProfiles[x].SourceProfile
            if sourceProfile == profileName {
                // grab username from MFA ARN
                userName  := strings.Split(awsRoleProfiles[x].MfaSerial, "/")[1]
                accountId := strings.Split(awsRoleProfiles[x].MfaSerial, ":")[4]
                awsAuthProfiles[x].UserName  = userName
                awsAuthProfiles[x].AccountId, err = strconv.Atoi(accountId)
                check(err)

                if sourceProfile == "default" {
                    portrayConfig.DefaultProfile   = "default"
                    portrayConfig.DefaultAccountId = awsAuthProfiles[x].AccountId
                }
                break
            }
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
	err = ioutil.WriteFile("test.yaml", yamlData, 0644)
	check(err)
    // dump json to file
	err = ioutil.WriteFile("test.json", jsonData, 0644)
	check(err)
}
	
func check(e error) {
	if e != nil {
		panic(e)
	}
}
