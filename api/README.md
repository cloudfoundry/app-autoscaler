## Run Unit Tests and Linter
Modify the test DB_URI in the test script in package.json
```sh
npm install
npm test
npm run lint
```

## Deploy the application to cloudfoundry
```sh
cf push autoscaler-api -f ../manifest.yml
```
