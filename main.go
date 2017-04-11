package main

import (
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
	"syscall"
	"time"
)

func main() {
	// Subcommands
	authCommand := flag.NewFlagSet("auth", flag.ExitOnError)
	switchCommand := flag.NewFlagSet("switch", flag.ExitOnError)

	// Auth Options
	accountNumberPtr := authCommand.String("account", "", "The amazon account of your MFA device")
	userNamePtr := authCommand.String("username", "", "The amazon username associated with your MFA device")
	tokenCodePtr := authCommand.String("token", "", "The OTP token to use")

	// Switch Options
	roleAccountNumberPtr := switchCommand.String("account", "", "The amazon account of your role")
	roleNamePtr := switchCommand.String("role", "", "The amazon role name associated to assume")

	// Verify that a subcommand has been provided
	// os.Arg[0] is the main command
	// os.Arg[1] will be the subcommand
	if len(os.Args) < 2 {
		fmt.Println("auth, switch or clear subcommand is required")
		os.Exit(1)
	}
	profile := os.Getenv("AWS_PROFILE")
	if len(profile) == 0 {
		profile = "default"
	}

	// Get our session file
	usr, err := user.Current()
	checkError(err)
	fileName := usr.HomeDir + "/.aws/portray-session-" + profile + ".json"
	roleFileName := ""
	// Switch on the subcommand
	// Parse the flags for appropriate FlagSet
	// FlagSet.Parse() requires a set of arguments to parse as input
	// os.Args[2:] will be all arguments starting after the subcommand at os.Args[1]
	switch os.Args[1] {
	case "auth":
		authCommand.Parse(os.Args[2:])
	case "switch":
		switchCommand.Parse(os.Args[2:])
		roleFileName = usr.HomeDir + "/.aws/portray-role-session-" + *roleAccountNumberPtr + "_" + *roleNamePtr + ".json"
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	if authCommand.Parsed() {
		fmt.Println("In Auth")
		awsCreds := getCredsFromFile(fileName)
		if awsCreds.SessionToken == "" || !validateSession(awsCreds) {
			awsCreds = getNewSession(*accountNumberPtr, *userNamePtr, *tokenCodePtr)
			writeSessionFile(awsCreds, fileName)
		}
		fmt.Printf("AWS CREDS: %+v", awsCreds)
		if awsCreds.SessionToken == "" || !validateSession(awsCreds) {
			fmt.Println("You need a valid session!")
			os.Exit(1)
		}
		sessionToEnvVars(awsCreds, *accountNumberPtr, "")
		startShell(*accountNumberPtr)
	} else if switchCommand.Parsed() {
		fmt.Println("In Switch")
		awsRoleCreds := getCredsFromFile(roleFileName)
		if awsRoleCreds.SessionToken == "" || !validateSession(awsRoleCreds) {
			awsCreds := getCredsFromFile(fileName)
			if awsCreds.SessionToken == "" || !validateSession(awsCreds) {
				fmt.Println("You need a valid session!")
				os.Exit(1)
			}
			awsRoleCreds = getNewRoleSession(*roleAccountNumberPtr, *roleNamePtr, *usr)
			writeSessionFile(awsRoleCreds, roleFileName)
		}
		sessionToEnvVars(awsRoleCreds, *roleAccountNumberPtr, *roleNamePtr)
		startShell(*roleAccountNumberPtr + "-" + *roleNamePtr)
	}
}

type AwsCreds struct {
	AccessKeyId     string
	Expiration      int64
	SecretAccessKey string
	SessionToken    string
}

func getNewSession(accountNumber string, userName string, tokenCode string) (awsCreds AwsCreds) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	checkError(err)
	fmt.Println("got AWS Session")
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
	}

	return
}

func getNewRoleSession(accountNumber string, roleName string, usr user.User) (awsCreds AwsCreds) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	checkError(err)
	fmt.Println("got AWS Session")
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
	}

	return
}

func writeSessionFile(awsCreds AwsCreds, fileName string) {
	awsCredsJson, _ := json.Marshal(awsCreds)

	createFile(fileName)
	err := ioutil.WriteFile(fileName, awsCredsJson, 0600)
	checkError(err)
	fmt.Println("Wrote session file")
}

func sessionToEnvVars(awsCreds AwsCreds, account string, role string) {
	fmt.Println("Setting ENV VARS")
	os.Setenv("AWS_ACCESS_KEY_ID", awsCreds.AccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", awsCreds.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", awsCreds.SessionToken)
	os.Setenv("PORTRAY_PROMPT", account+":"+role)

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
	fmt.Println("Checking valid session")
	valid = false
	timestamp := int64(time.Now().Unix())
	if timestamp < awsCreds.Expiration {
		valid = true
	}
	fmt.Printf("Valid: %t", valid)
	return
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
