# Lightsail-lockdown

Lightsail-lockdown is a trivial utility to modify existing port access (firewall) rules on AWS Lightsail instances to limit external access to specific CIDR address blocks.

The intended use case is to allow users with dynamic IP addresses to protect Lightsail instances which do not need to accept incoming connections from the open Internet. This scenario is most likely applicable to Lightsail instances used for educational purposes or as development/test environments.

Lightsail-lockdown can be called by [atrack](https://github.com/coridonhenshaw/atrack) to perform automatic updates in dynamic IP environments.

# Requirements

Appropriate credentials to access the AWS API must be present in ~/.aws/credentials. See the AWS SDK documentation for instructions on how to [obtain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/setting-up.html#get-aws-credentials) and [install](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) credentials.

**Follow AWS best practices (such as using IAM credentials with limited access rights) to ensure that AWS credentials which could cause financial losses if misused are adequately protected.**

# Limitations

There is no provision to allow any ports to remain accessible to the public.

# Usage

    lockdown -r <region> -i <instance> [-d] [-f] [-4 <IPV4 CIDR>] [-6 <IPV6 CIDR>]

##### Where:

Parameter | Description
-|-
-i \<region\> | The name of the AWS region containing the Lightsail instance to protect. Required.
-i \<instance\> | The name of the AWS Lightsail instance to protect. Required.
-4 \<IPV4 CIDR\> | IPv4 CIDR mask from which access is to be allowed to the Lightsail instance. Specify 'none' to clear the existing CIDR mask.
-6 \<IPV6 CIDR\> | IPv6 CIDR mask from which access is to be allowed to the Lightsail instance. Specify 'none' to clear the existing CIDR mask.
-d | Dry-run: do everything but send the firewall update to the AWS API.
-f | Force update even if no changes are required.

## Platform Compatibility

Lightsail-lockdown is built on Linux (specifically: OpenSUSE) but should build on any platform supported by Go.

# Release history

0 - Initial release in Python.

1 - Complete rewrite in Golang to remove external dependencies on the AWS CLI tool.

2 - Updated to use Go modules. Improved handling of Lightsail firewall configurations where different rules are applied to IPv4 and IPv6 connections. Minor code rework for readability.

# License

Copyright 2021, 2025 Coridon Henshaw

Permission is granted to all natural persons to execute, distribute, and/or modify this software (including its documentation) subject to the following terms:

1. Subject to point \#2, below, **all commercial use and distribution is prohibited.** This software has been released for personal and academic use for the betterment of society through any purpose that does not create income or revenue. *It has not been made available for businesses to profit from unpaid labor.*

2. Re-distribution of this software on for-profit, public use, repository hosting sites (for example: Github) is permitted provided no fees are charged specifically to access this software.

3. **This software is provided on an as-is basis and may only be used at your own risk.** This software is the product of a single individual's recreational project. The author does not have the resources to perform the degree of code review, testing, or other verification required to extend any assurances that this software is suitable for any purpose, or to offer any assurances that it is safe to execute without causing data loss or other damage.

4. **This software is intended for experimental use in situations where data loss (or any other undesired behavior) will not cause unacceptable harm.** Users with critical data safety needs must not use this software and, instead, should use equivalent tools that have a proven track record.

5. If this software is redistributed, this copyright notice and license text must be included without modification.

6. Distribution of modified copies of this software is discouraged but is not prohibited. It is strongly encouraged that fixes, modifications, and additions be submitted for inclusion into the main release rather than distributed independently.

7. This software reverts to the public domain 10 years after its final update or immediately upon the death of its author, whichever happens first.
