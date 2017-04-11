package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Subcommands
	authCommand := flag.NewFlagSet("auth", flag.ExitOnError)
	switchCommand := flag.NewFlagSet("switch", flag.ExitOnError)

	// Auth Options
	profilePtr := authCommand.String("profile", "", "The amazon credentials profile")
	accountNumberPtr := authCommand.String("account", "", "The amazon account of your MFA device")
	userNamePtr := authCommand.String("username", "", "The amazon username associated with your MFA device")
	tokenCodePtr := authCommand.String("token", "", "The OTP token to use")
	savePtr := authCommand.Bool("save", false, "Save the supplied details")

	// Switch Options
	profilePtr = switchCommand.String("profile", "", "The amazon credentials profile")
	roleAccountNumberPtr := switchCommand.String("account", "", "The amazon account of your role")
	roleNamePtr := switchCommand.String("role", "", "The amazon role name associated to assume")
	saveProfilePtr := switchCommand.Bool("save", false, "Save the supplied details")

	// Verify that a subcommand has been provided
	// os.Arg[0] is the main command
	// os.Arg[1] will be the subcommand
	if len(os.Args) < 2 {
		fmt.Println("auth, switch or clear subcommand is required")
		os.Exit(1)
	}

	profile := "default"
	envProfile := os.Getenv("AWS_PROFILE")
	fileName := ""
	roleFileName := ""

	// Get our session file
	usr, err := user.Current()
	checkError(err)
	portrayConfigFileName := usr.HomeDir + "/.portray-config.json"
	config := getPortrayConfigFromFile(portrayConfigFileName)

	// Switch on the subcommand
	// Parse the flags for appropriate FlagSet
	// FlagSet.Parse() requires a set of arguments to parse as input
	// os.Args[2:] will be all arguments starting after the subcommand at os.Args[1]
	switch os.Args[1] {
	case "auth":
		authCommand.Parse(os.Args[2:])
		if *profilePtr != "" {
			profile = *profilePtr
			os.Setenv("AWS_PROFILE", profile)
		} else if envProfile != "" {
			profile = envProfile
		} else if config.DefaultProfile != "" {
			profile = config.DefaultProfile
		}
		fileName = usr.HomeDir + "/.aws/portray-session-" + profile + ".json"
	case "switch":
		switchCommand.Parse(os.Args[2:])
		if *profilePtr != "" {
			profile = *profilePtr
			os.Setenv("AWS_PROFILE", profile)
		} else if envProfile != "" {
			profile = envProfile
		} else if config.DefaultProfile != "" {
			profile = config.DefaultProfile
		}
		fileName = usr.HomeDir + "/.aws/portray-session-" + profile + ".json"
		roleFileName = usr.HomeDir + "/.aws/portray-role-session-" + *roleAccountNumberPtr + "_" + *roleNamePtr + ".json"
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	if authCommand.Parsed() {
		awsCreds := getCredsFromFile(fileName)
		if awsCreds.SessionToken == "" || !validateSession(awsCreds) {
			if *tokenCodePtr != "" {
				awsCreds = getNewSession(*accountNumberPtr, *userNamePtr, *tokenCodePtr)
			} else if config.AccountNumber != "" {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter token: ")
				token, _ := reader.ReadString('\n')
				token = strings.TrimSpace(token)
				awsCreds = getNewSession(config.AccountNumber, config.UserName, token)
			} else {
				fmt.Println("You need a valid session!")
				os.Exit(1)
			}
			writeSessionFile(awsCreds, fileName)
		}
		if awsCreds.SessionToken == "" || !validateSession(awsCreds) {
			fmt.Println("You need a valid session!")
			os.Exit(1)
		}

		accountNumber := awsCreds.AccountNumber

		sessionToEnvVars(awsCreds, accountNumber, "", profile)
		if *savePtr == true {
			if *profilePtr != "" {
				config.DefaultProfile = *profilePtr
			}
			if *accountNumberPtr != "" {
				config.AccountNumber = *accountNumberPtr
			}

			if *userNamePtr != "" {
				config.UserName = *userNamePtr
			}
			writePortrayConfigToFile(portrayConfigFileName, config)
		}
		startShell(accountNumber)
	} else if switchCommand.Parsed() {
		profileIdx := -1

		if *profilePtr != "" {
			for i := range config.Profiles {
				if config.Profiles[i].Name == *profilePtr {
					profileIdx = i
					fmt.Println("Found Profile")
					roleFileName = usr.HomeDir + "/.aws/portray-role-session-" + config.Profiles[i].AccountNumber + "_" + config.Profiles[i].RoleName + ".json"
					break
				}
			}
		}

		awsRoleCreds := getCredsFromFile(roleFileName)
		if awsRoleCreds.SessionToken == "" || !validateSession(awsRoleCreds) {
			fileName = usr.HomeDir + "/.aws/portray-session-" + envProfile + ".json"
			awsCreds := getCredsFromFile(fileName)
			if awsCreds.SessionToken == "" || !validateSession(awsCreds) {
				if config.AccountNumber != "" {
					fmt.Printf("Attempt auth using config...")
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("Enter token: ")
					token, _ := reader.ReadString('\n')
					token = strings.TrimSpace(token)
					awsCreds = getNewSession(config.AccountNumber, config.UserName, token)
					writeSessionFile(awsCreds, fileName)
				} else {
					fmt.Println("You need a valid session!")
					os.Exit(1)
				}
			}
			if *roleAccountNumberPtr == "" && profileIdx == -1 {
				fmt.Println("You must have a profile configured or supply a role account #")
				os.Exit(1)
			}
			if *roleAccountNumberPtr != "" {
				awsRoleCreds = getNewRoleSession(*roleAccountNumberPtr, *roleNamePtr, *usr)
			} else {
				awsRoleCreds = getNewRoleSession(
					config.Profiles[profileIdx].AccountNumber,
					config.Profiles[profileIdx].RoleName,
					*usr,
				)
			}
			writeSessionFile(awsRoleCreds, roleFileName)
		}

		roleAccountNumber := awsRoleCreds.AccountNumber
		roleName := awsRoleCreds.RoleName

		sessionToEnvVars(awsRoleCreds, roleAccountNumber, roleName, profile)
		if *saveProfilePtr == true {
			if profileIdx >= 0 {
				config.Profiles[profileIdx].RoleName = *roleNamePtr
				config.Profiles[profileIdx].AccountNumber = *roleAccountNumberPtr
				config.Profiles[profileIdx].Name = *profilePtr
			} else if *profilePtr != "" {
				profile := PortrayProfile{*profilePtr, *roleAccountNumberPtr, *roleNamePtr}
				config.Profiles = append(config.Profiles, profile)
			}
			writePortrayConfigToFile(portrayConfigFileName, config)
		}
		startShell(roleAccountNumber + "-" + *roleNamePtr)
	}
}

type AwsCreds struct {
	AccessKeyId     string
	Expiration      int64
	SecretAccessKey string
	SessionToken    string
	AccountNumber   string
	RoleName        string
}

type PortrayProfile struct {
	Name          string
	AccountNumber string
	RoleName      string
}

type PortrayConfig struct {
	DefaultProfile string
	AccountNumber  string
	UserName       string
	Profiles       []PortrayProfile
}

func getNewSession(accountNumber string, userName string, tokenCode string) (awsCreds AwsCreds) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	checkError(err)
	svc := sts.New(sess)

	params := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(43200),
		SerialNumber:    aws.String("arn:aws:iam::" + accountNumber + ":mfa/" + userName),
		TokenCode:       aws.String(tokenCode),
	}

	resp, err := svc.GetSessionToken(params)

	checkError(err)

	awsCreds = AwsCreds{*resp.Credentials.AccessKeyId,
		resp.Credentials.Expiration.Unix(),
		*resp.Credentials.SecretAccessKey,
		*resp.Credentials.SessionToken,
		accountNumber,
		"",
	}

	return
}

func getNewRoleSession(accountNumber string, roleName string, usr user.User) (awsCreds AwsCreds) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	checkError(err)
	svc := sts.New(sess)

	timestamp := int64(time.Now().Unix())

	params := &sts.AssumeRoleInput{
		ExternalId:      aws.String(usr.Username),
		DurationSeconds: aws.Int64(3600),
		RoleArn:         aws.String("arn:aws:iam::" + accountNumber + ":role/" + roleName),
		RoleSessionName: aws.String(roleName + "-" + usr.Username + "-" + strconv.FormatInt(timestamp, 10)),
	}

	resp, err := svc.AssumeRole(params)

	checkError(err)

	awsCreds = AwsCreds{*resp.Credentials.AccessKeyId,
		resp.Credentials.Expiration.Unix(),
		*resp.Credentials.SecretAccessKey,
		*resp.Credentials.SessionToken,
		accountNumber,
		roleName,
	}

	return
}

func writeSessionFile(awsCreds AwsCreds, fileName string) {
	awsCredsJson, _ := json.Marshal(awsCreds)

	createFile(fileName)
	err := ioutil.WriteFile(fileName, awsCredsJson, 0600)
	checkError(err)
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
	os.Setenv("AWS_PROFILE", profile)
	os.Setenv("AWS_ACCESS_KEY_ID", awsCreds.AccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", awsCreds.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", awsCreds.SessionToken)
	os.Setenv("PORTRAY_PROMPT", prompt)

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

func validateSession(awsCreds AwsCreds) (valid bool) {
	valid = false
	timestamp := int64(time.Now().Unix())
	if timestamp < awsCreds.Expiration {
		valid = true
	}
	return
}

func getPortrayConfigFromFile(fileName string) (config PortrayConfig) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return
	}

	json.Unmarshal(file, &config)
	return
}

func writePortrayConfigToFile(fileName string, config PortrayConfig) {
	configJson, _ := json.Marshal(config)

	createFile(fileName)
	err := ioutil.WriteFile(fileName, configJson, 0600)
	checkError(err)
}

func getCredsFromFile(fileName string) (awsCreds AwsCreds) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return
	}

	json.Unmarshal(file, &awsCreds)
	return
}

func startShell(account string) {
	fmt.Println("Starting shell with Session in: " + account)
	syscall.Exec(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, syscall.Environ())
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
