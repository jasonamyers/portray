# portray
Portray is meant to allow you to portray yourself as another user on AWS. This is useful if
 you have AWS credentials for an AWS account protected by MFA, and you use that AWS account
  to assume roles in other AWS accounts. You must have a set of credentials stored in
  ``~/.aws/credentials`` in order to start using portray. You can learn more in the [AWS CLI Config Guide](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

## Authenticating
Authenticating with portray is done via the auth subcommand. It requires an AWS account
number, AWS username, and a OTP token code. (You can optionally define everything but the
token code in the [portray config file](#config).) It starts an Amazon STS session valid
for 12 hours, and stores it in a file in the ``~/.aws/`` directory. It also starts a new
shell in which this session is activated and exported as environment variable. Details about the
session can be added to the  prompt as describe in the [Prompt](#prompt) section. A simple exit
of this new shell will return you to your prior state.

### Starting a session explicitly
The command below is an example of how to start a session:

``portray auth --account <aws_account_number> --username <aws_user_name> --token <otp_token_code>``

You can optionally specify an AWS profile to use in your AWS Credentials file by adding
``--profile <aws_creds_profile_name>`` onto the command above(currently broken).  You can also save all the
supplied details as the defaults for the auth command by appending ``--save`` onto the
command.  This will store the account, username, and profile so you don't have to specify
them each time. More details on that are available in the [Config](#config) section

### Starting a session from config
The command below is an example of how to start a session based on config.

``portray auth``

NOTE: This will prompt you for the token code, and it should not be supplied on the command
line.


## Switch to another role
Another use of portray is to switch to a role in another account to which you are
authorized to assume. It starts an Amazon STS session valid for 1 hour, and stores it in a file in the ``~/.aws/``
directory. It also starts a new shell in which this session is activated. Details about the session can be added to the
prompt as describe in the [Prompt](#prompt) section. A simple exit of this new shell will return you to your prior state.
 If you do not have a valid authenticated session, you will be prompted for a token if you have a
 saved configuration or told to try again if a saved configuration is not available.

### Switch to a role in a particular account
The example below is an example of how to switch to another role:

``portray switch --account <aws_account_number> --role <aws_role_name>``

It is possible to provide an alias for the account and role with the ``--profile <profile_name>``
argument. It is possible to save the details of a role into the [portray config file](#config) by providing a
using the ``--save`` flag. You must supply a profile name in order to save the role details.

### Switch to a role from config
Here is an example of using a saved role from configuration in the switch

``./portray switch --profile dev``

## Config
Portray stores its configuration in the ``~/.portray-config.json`` file as a JSON object. It
consists of some defaults for authenticating (Default AWS Credentials Profile, Account Number, Username),
and a list of Profiles to use when switching to other accounts/roles.

You can see an example here.

```JSON
{
	"DefaultProfile": "",
	"AccountNumber": "123456789012",
	"UserName": "user.name",
	"Profiles": [{
		"Name": "prod",
		"AccountNumber": "234567890123",
		"RoleName": "SomeRole"
	}, {
		"Name": "dev",
		"AccountNumber": "345678901234",
		"RoleName": "SomeRole"
	}]
}
```

## Prompt

Portray updates a $PORTRAY_PROMPT environment variable with an account number, role name, and profile so
you can include that in your prompt to have that information available in your prompt.

For example after an auth:

``123456789012:ctl [jasonamyers:~/dev/portray] master(+92/-12)* ± exit``

After a switch

``234567890123:Admin:dev [jasonamyers:~/dev/portray] master(+92/-12)* ± exit``

## Developing

`dep` is used for package management. Use `dep ensure` to keep Gopkg.lock and
vendored packages in sync.
