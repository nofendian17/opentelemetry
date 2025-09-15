#!/bin/bash

BASE_URL="http://localhost:8080"

echo "== Root endpoint =="
curl -i "$BASE_URL/"

echo -e "\n\n== Health endpoint =="
curl -i "$BASE_URL/health"

echo -e "\n\n== List users =="
curl -i "$BASE_URL/users"

echo -e "\n\n== Create 1000 random users =="
for i in {1..100000}; do
  name="User_$RANDOM"
  email="user_$RANDOM@example.com"
  curl -i -X POST "$BASE_URL/users" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$name\",\"email\":\"$email\"}"
done

echo -e "\n\n== Get user by ID =="
curl -i "$BASE_URL/users/1"

echo -e "\n\n== Update user =="
curl -i -X PUT "$BASE_URL/users/1" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Updated","email":"alice@example.com"}'

echo -e "\n\n== Delete user =="
curl -i -X DELETE "$BASE_URL/users/1"
