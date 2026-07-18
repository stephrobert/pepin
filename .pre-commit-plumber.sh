#!/bin/sh
# Plumber audit des workflows GitHub Actions — nécessite GITLAB_TOKEN ;
# ignoré silencieusement s'il est absent.
if [ -z "$GITLAB_TOKEN" ]; then
  echo "plumber: GITLAB_TOKEN absent, audit ignoré"
  exit 0
fi
exec plumber analyze --threshold 100
