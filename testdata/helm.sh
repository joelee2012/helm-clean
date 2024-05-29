#!/bin/bash
if [ $1 == "list" ]; then
  cat <<EOF
[
  {
    "name": "appserver-home",
    "namespace": "tsp",
    "revision": "498",
    "updated": "2024-05-29T10:58:00",
    "status": "deployed",
    "chart": "backend-service-2.0.2",
    "app_version": "1.16.0"
  },
  {
    "name": "tboxsimulator-home",
    "namespace": "tsp",
    "revision": "81",
    "updated": "2024-04-23T21:00:43",
    "status": "deployed",
    "chart": "backend-service-2.0.2",
    "app_version": "1.16.0"
  },
  {
    "name": "tsp",
    "namespace": "tsp",
    "revision": "250",
    "updated": "2024-05-29T10:37:14",
    "status": "deployed",
    "chart": "web-service-0.1.3",
    "app_version": "1.16.0"
  },
  {
    "name": "tspadapter-home",
    "namespace": "tsp",
    "revision": "174",
    "updated": "2024-05-21T14:34:54",
    "status": "deployed",
    "chart": "backend-service-2.0.2",
    "app_version": "1.16.0"
  },
  {
    "name": "tspbackend-home",
    "namespace": "tsp",
    "revision": "1235",
    "updated": "2024-05-28T15:54:20",
    "status": "deployed",
    "chart": "web-service-2.0.2",
    "app_version": "1.16.0"
  }
]
EOF
elif [ "$1" == "uninstall" ]; then
  echo "$@"
else
  echo "unkonwn args: $@"
  exit 1
fi
