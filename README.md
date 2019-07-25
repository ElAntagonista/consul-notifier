# Consul Notifier

## Table of Contents

- [About](#about)
- [Usage](#usage)


## About <a name = "about"></a>

This is a simple implementation of a Consul watch http handler written in Go that sends a message in a Slack channel on a given change in a check object.
Currently this is just a PoC so I highly discourage using it in a production environment.
### Installing

Clone/Download this repo. Run ``` go build ``` in the main directory. 

## Usage <a name = "usage"></a>

consul-notifier -slackurl \<your-incoming-webhook-url>

