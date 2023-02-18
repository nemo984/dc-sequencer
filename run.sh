#!/bin/bash

function cleanup {
    pkill -SIGINT -f "go run"
    echo "Output of each members:"
    ls | grep 'output-' | while read filename; do
        echo $filename 
    done
    exit
}

trap cleanup SIGINT

rm output-*.txt &> /dev/null

NUM_INSTANCES="${1:-2}"

echo "Running 1 instance of sequencer, and $NUM_INSTANCES instances of member, Ctrl+C to stop"

go run cmd/sequencer/main.go &

for (( i=1; i<=NUM_INSTANCES; i++ ))
do
    go run cmd/member/main.go &
done

wait
