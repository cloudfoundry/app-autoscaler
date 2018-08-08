## Run Unit Tests and Linter
Modify the DB_URI and the SCHEDULER_URI within the test script in package.json to appropriate values
```sh
npm install
npm run lint
npm test
```

## Deploy the application to cloudfoundry
```sh
cf push autoscaler-api -f ../manifest.yml
```