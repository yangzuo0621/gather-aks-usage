#!/usr/bin/env bash

## gather the information every 30 minutes
while true; do
  echo "Gather information..."
  ./gather-aks-usage > output
  sleep 1800
done