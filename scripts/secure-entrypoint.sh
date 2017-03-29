#!/bin/bash

eval $(/usr/sbin/secure-environment export)

exec "$@"
