#!/bin/bash

 ssh ubuntu@blog.cirias.li 'cd bookkeeper && docker-compose stop'
 scp ./bookkeeper ubuntu@blog.cirias.li:~/bookkeeper/
 ssh ubuntu@blog.cirias.li 'cd bookkeeper && docker-compose restart'
