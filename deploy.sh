#!/bin/bash

set -e

STORE=$1
PASSWORD=$2

go build

secmuxer -store $STORE -password $PASSWORD ./credentials.json.tpl > credentials.json
secmuxer -store $STORE -password $PASSWORD ./docker-compose.yml.tpl > docker-compose.yml

scp ./bookkeeper ubuntu@blog.cirias.li:~/bookkeeper/
scp ./credentials.json ubuntu@blog.cirias.li:~/bookkeeper/
scp ./docker-compose.yml ubuntu@blog.cirias.li:~/bookkeeper/

ssh ubuntu@blog.cirias.li 'cd bookkeeper && docker-compose up -d --force-recreate --remove-orphans'
