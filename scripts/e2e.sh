#!/bin/bash

test_runner="go test ./e2e -v -count=1 \
        -debug=true \
        -host=localhost \
        -port=8080 \
        -target-url=http://jsonplaceholder.typicode.com/ \
        -body-methods-only=true \
        -reject-with=bad_message \
        -reject-exact=true \
        -reject-insensitive=false \
        -request-delay=2"

file=.env
if [ -e "$file" ]; then
    echo ".env file exists, renaming it so that PROD configs are not used to run e2e tests"
    mv .env temp_config_name
    eval $test_runner
    echo "undoing name change of .env file"
    mv temp_config_name .env

else 
    echo ".env file does not exists, running e2e tests"
    eval $test_runner
    
fi 
    echo "e2e tests complete"
