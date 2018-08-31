#!/bin/bash
response=$(curl -H "Host:test.localhost:3999" 0.0.0.0:3999 --write-out "%{http_code}\n" --silent --output /dev/null)
if [ "$response" -eq 200 ];then 
    echo "$(tput setaf 2)Passed$(tput sgr0) routing correct http url"
else
    echo "$(tput setaf 1)Failed$(tput sgr0) routing correct http url"
fi

response=$(curl -H "Host:fake.localhost:3999" 0.0.0.0:3999 --write-out "%{http_code}\n" --silent --output /dev/null)
if [ "$response" -ne 200 ];then 
    echo "$(tput setaf 2)Passed$(tput sgr0) routing incorrect http url"
else
    echo "$(tput setaf 1)Failed$(tput sgr0) routing incorrect http url"
fi
response=$(curl -k --resolve test.localhost:4000:127.0.0.1 https://test.localhost:4000 --write-out "%{http_code}\n" --silent --output /dev/null)
if [ "$response" -eq 200 ];then 
    echo "$(tput setaf 2)Passed$(tput sgr0) routing correct https url"
else
    echo "$(tput setaf 1)Failed$(tput sgr0) routing correct https url"
fi

response=$(curl -k --resolve fake.localhost:4000:127.0.0.1 https://fake.localhost:4000 --write-out "%{http_code}\n" --silent --output /dev/null)
if [ "$response" -ne 200 ];then 
    echo "$(tput setaf 2)Passed$(tput sgr0) routing incorrect https url"
else
    echo "$(tput setaf 1)Failed$(tput sgr0) routing incorrect https url"
fi