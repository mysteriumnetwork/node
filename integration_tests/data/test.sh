#!/usr/bin/env bash

output=`/usr/bin/mysterium_client --cli < /usr/lib/integration_tests/cli-input`

echo "Full CLI output:"
echo "$output"
echo

status_line=`echo "$output" | grep "Status: NotConnected"`
if [ -z "$status_line" ]
then
	echo "Status line not found"
	exit 1
fi
echo "Status line found:"
echo $status_line

echo "Tests passed"
