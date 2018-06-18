#!/usr/bin/env bash

#don't connect to management interface - fail early (simulating invalid args combination)

echo "Error: Unsupported args: $@"
exit 1