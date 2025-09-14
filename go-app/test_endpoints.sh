#!/bin/bash

BASE_URL="http://localhost:8080"

echo "== Root endpoint =="
curl -i "$BASE_URL/"

echo -e "\n\n== Health endpoint =="
curl -i "$BASE_URL/health"

echo -e "\n\n== List users =="
curl -i "$BASE_URL/users"

echo -e "\n\n== Create user =="
curl -i -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'

echo -e "\n\n== Get user by ID =="
curl -i "$BASE_URL/users/1"

echo -e "\n\n== Update user =="
curl -i -X PUT "$BASE_URL/users/1" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Updated","email":"alice@example.com"}'

echo -e "\n\n== Delete user =="
curl -i -X DELETE "$BASE_URL/users/1"
