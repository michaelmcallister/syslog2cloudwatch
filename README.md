# syslog2cloudwatch

This is a very very early commit of a Syslog reciever that ships syslog messages straight to AWS CloudWatch

This will (hopefully) change rapidly, with more features, better error handling - and way more configurability

I'm still learning golang so I would definitely appreciate any tips if you see any oddities.

## What works:
- basic config file to specify log group and log stream
- receiving syslog messages and...
- sending them to CloudWatch (duh)

## What doesn't:
- the syslog format is hardcoded, so it's not very flexible
- error handling isn't the best - most of the time it'll print to STDERR, but it may silently fail too
- automatically creating the log group and/or log stream if it doesn't exist

## TODO:
- way more options to configure
- handle replicating AWS config into my config file (access keys, region config)
