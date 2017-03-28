# secure-environment - A loader for secure environments on AWS

## Introduction

This tool is intended to be used on the start up of a docker container to
securely set environment variables on startup of a container. The included
`secure-entrypoint.sh` script is intended to be used along with the
`secure-environment` to provide this functionality on docker containers. At
this time, this is intended to work with [convox](https://convox.com)
specifically.

### How it works

The `docker-entrypoint.sh` script acts as an entrypoint for the docker
container. The script then calls the `secure-environment` binary which will
then write a sourceable shell script to stdout that contains `export`ed
environment variables.

## Using with convox

### Setting up the docker container

To use this with convox, you need to set the label `convox.secure-env` to true
on the services you intend to secure. 

On your docker container you will want to make sure that the
`secure-entrypoint.sh` in the scripts folder of this repository and the latest
linux binary of the `secure-environment` executable are copied into your docker
container to the following locations:

```
secure-environment -> /secure-environment
secure-entrypoint -> /secure-entrypoint.sh
```

_If you know what you're doing you can update the `secure-entrypoint.sh` file so you can change the location of these files._

Finally, you need to set the `ENTRYPOINT` on your dockerfile to this:

```
ENTRYPOINT ["/secure-entrypoint.sh"]
```

If you're using this with tini like we do at Virtru, then you would do this:

```
ENTRYPOINT ["/usr/local/bin/tini", "--", "/secure-entrypoint.sh"]
```
