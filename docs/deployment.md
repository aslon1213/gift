# This Guide is about guidelines for deploying Gift server on different platforms

### On VPS
ssh to any VPS
clone the repository
```sh
git clone https://github.com/aslon1213/gift.git
cd gift
```

```
curl https://mise.run | sh
mise install
```

```
echo 'DB_URL=<mongo_db_database_url>' > server/.env
```

```
mise run api
```

```
mise run web
```

### Deploy Using Containers
By this approach can be deployed to any platform that supports containers.
dockerfiles are inside deployment folder, there is also docker compose file which has mongodb as well;


### Deploy on Railway
Connect this repo to railway and set root path to /server for api deployment and root path to /client/gift for web deployment.
Port: 3000 - api, 5173 - web
