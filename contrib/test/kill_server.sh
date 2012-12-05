#!/bin/bash

ps aux | grep echoserver | awk '/atrejo/' | awk '!/grep/ { print $2 | "xargs kill -9"; exit }'
