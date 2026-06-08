#!/bin/bash
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"1HBZbdawJ5i5dMFso1FW"}' \
  http://localhost:8443/api/auth/login
