'use strict';

var path = require('path');
var cp = require('child_process');
var fs = require('fs');
var dbConnectionInfo = require(path.join(__dirname, '../../../lib/util/dbConnectionInfo.js'));

var initDB = function initDB() {
  var dbServer = dbConnectionInfo.dbServer;
  var dbName = dbConnectionInfo.dbName;
  var psqlConnectionStr = 'psql '+ dbServer;

  var dropDBCmd = psqlConnectionStr + ' -c "DROP DATABASE IF EXISTS ' + dbName + ';"';
  var createDBCmd = psqlConnectionStr + ' -c "CREATE DATABASE ' + dbName + ';"';
  var createTableCmd = psqlConnectionStr + '/' + dbName + ' -c "' + 'CREATE TABLE service_instance(service_id character varying(40) NOT NULL,org_id character varying(40) NOT NULL,space_id character varying(40) NOT NULL,CONSTRAINT service_instance_pkey PRIMARY KEY (service_id));' + 'CREATE INDEX ix_service_instance ON service_instance USING btree (service_id , org_id , space_id);' + 'CREATE TABLE binding(app_id character varying(40) NOT NULL,service_id character varying(40) NOT NULL,binding_id character varying(40) NOT NULL,"timestamp" bigint NOT NULL,CONSTRAINT bingding_pkey PRIMARY KEY (binding_id),CONSTRAINT fk_binding_serviceid FOREIGN KEY (service_id) REFERENCES service_instance (service_id) ON UPDATE NO ACTION ON DELETE NO ACTION,CONSTRAINT ux_binding UNIQUE (binding_id, app_id, service_id));' + 'CREATE INDEX ix_binding ON binding USING btree (binding_id);"';

  cp.exec(dropDBCmd, function(error, stdout, stderr) {
    console.log('stdout: ' + stdout);
    console.log('stderr: ' + stderr);
    if (error !== null) {
      console.log('exec error: ' + error);
    } else {
      cp.exec(createDBCmd, function(error, stdout, stderr) {
        console.log('stdout: ' + stdout);
        console.log('stderr: ' + stderr);
        if (error !== null) {
          console.log('exec error: ' + error);
        } else {
          cp.exec(createTableCmd, function(error, stdout, stderr) {
            console.log('stdout: ' + stdout);
            console.log('stderr: ' + stderr);
            if (error !== null) {
              console.log('exec error: ' + error);
            } else {
              console.log('init database ' + dbName  + ' successfully!');
            }
          });
        }
      });
    }
  });
};
initDB();