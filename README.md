# portray
Portray is meant to allow you to portray yourself as another user on AWS.

Start a STS session via MFA

``./main auth --account <aws_account_number> --username <aws_user_name> --token <otp_token_code>``

Switch to a role in a particular account

``./main switch --account <aws_account_number> --role <aws_role_name>``
