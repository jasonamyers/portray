# Portray

Portray is meant to allow you to portray yourself as another user on AWS. This
is useful if you have AWS credentials that you use to assume roles,
particularly if role assumption requires a valid MFA session.

You must have a set of credentials stored in `~/.aws/credentials` in order to
start using Portray. You can learn more in the [AWS CLI Config Guide](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).
Portray can parse the AWS CLI config and expects it to be located at
`~/.aws/config`. See the [Config](#config) section for more info.

## Authenticating

Authenticating with portray is done via the auth subcommand. It requires an AWS
account number, AWS username, and a OTP token code. You can optionally define
everything but the token code in the [Portray config file](#config). It starts
an Amazon STS session that's valid for 12 hours and caches the credentials in
in a file in the `~/.aws/` directory. It also starts a new shell where these
credentials are exported as environment variables. Details about the session
can be added to the prompt as described in the [Prompt](#prompt) section. A
simple exit of this new shell will return you to your prior state.

### Starting a session explicitly

The command below is an example of how to start a session:

`portray auth --account <aws_account_number> --username <aws_user_name> --token <otp_token_code>`

If you don't supply the token code via and don't specify `--no-mfa`, you'll be
prompted for the token.

### Starting a session from config

Starting sessions from config uses the named AuthProfiles. If no arguments are
passed, it will use Portray's configured "default" profile.

`portray auth`

To use a named profile, pass the `--profile` flag:

`portray switch --profile dev`

Both of these commands will prompt you for the MFA token if it's not supplied
via the `--token` flag.

## Switching Roles

Another use of Portray is switching AWS roles. These roles can be in the same
account as the AuthProfile, or they can be another account where you're
authorized to assume roles.

The switch command starts an STS session that's valid for 1 hour and caches the
credentials in a file in the `~/.aws/` directory.

It also starts a new shell where these credentials are exported as environment
variables. Details about the session can be added to the prompt as described in
the [Prompt](#prompt) section. A simple exit of this new shell will return you
to your prior state.

### Switch to a role explicitly

The command below is an example of how to assume a role:

`portray switch --account <aws_account_number> --role <aws_role_name>`

### Switch to a role from config

Here is an example of using a saved role from configuration in the switch

``./portray switch --profile dev``

Starting sessions from config uses the named Profiles.

## Config

By default, Portray reads its configuration from `~/.portray.yaml`.

The recommended way of populating this configuration is by parsing the AWS CLI
config with `portray config --sync`. This will output the YAML config to stdout
by default. See `portray config -h` for more options.

The Portray configuration consists of two sections/concepts: AuthProfiles and
Profiles. AuthProfiles are those where you would start an STS session via the
`portray auth` command, and Profiles are role profiles where you would assume a
role via the `portray switch` command.

You can see an example here.

```yaml
AuthProfiles:
  default:
    AccountId: "111111111111"
    Name: default
    Output: json
    Region: us-east-1
    UserName: user.name
  dev:
    AccountId: "222222222222"
    Name: dev
    Output: json
    Region: us-east-1
    UserName: user.name
Profiles:
  Admin:
    ExternalId: ""
    MfaSerial: arn:aws:iam::111111111111:mfa/user.name
    Name: Admin
    RoleArn: arn:aws:iam::111111111111:role/Admin
    RoleName: Admin
    SourceProfile: default
  DevAdmin:
    ExternalId: ""
    MfaSerial: arn:aws:iam::222222222222:mfa/user.name
    Name: DevAdmin
    RoleArn: arn:aws:iam::222222222222:role/Admin
    RoleName: Admin
    SourceProfile: dev
  SuperDevAdmin:
    ExternalId: ""
    MfaSerial: arn:aws:iam::111111111111:mfa/user.name
    Name: SuperDevAdmin
    RoleArn: arn:aws:iam::333333333333:role/Admin
    RoleName: Admin
    SourceProfile: default
```

Since Portray uses the [viper toolkit](https://github.com/spf13/viper) for
parsing configuration, it also supports a JSON config file. This can be
generated like the YAML config with `portray config --sync --format json`.

## Prompt

Portray adds a $PORTRAY_PROMPT environment variable with an account number,
role name, and profile so you can add that to your prompt.

For example after an auth:

``123456789012:ctl [jasonamyers:~/dev/portray] master(+92/-12)* ± exit``

After a switch

``234567890123:Admin:dev [jasonamyers:~/dev/portray] master(+92/-12)* ± exit``

## Developing

`dep` is used for package management. Use `dep ensure` to keep Gopkg.lock and
vendored packages in sync.
