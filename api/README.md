## Run Unit Tests and Linter
```sh
npm install
npm test
npm run lint
```

## Deploy the application to cloudfoundry
```sh
cf push autoscaler-api -f ../manifest.yml
```
