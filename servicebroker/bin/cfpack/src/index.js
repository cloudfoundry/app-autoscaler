'use strict';

// Package an app and its local npm dependencies for deployment to Cloud Foundry

// Implemented in ES5 for now
/* eslint no-var: 0 */

var _ = require('underscore');
var fs = require('fs');
var path = require('path');
var cp = require('child_process');
var commander = require('commander');

var map = _.map;
var pairs = _.pairs;
var object = _.object;
var extend = _.extend;
var partial = _.partial;
var noop = _.noop;

/* eslint no-process-exit: 1 */

// Create the directories we need
var mkdirs = function(root, cb) {
  // Create cfpackage directory
  fs.mkdir('cfpackage', function(err) {
    if(err) noop();
    fs.unlink('cfpackage/lib', function(err) {
      if(err) noop();
      fs.symlink(path.join(root, 'lib'), 'cfpackage/lib', cb);
    });
  });
};

// Adjust a file: dependency to our packed app structure, by converting
// relative path from the module to a relative path to our package root
var local = function(root, d) {
  return !/file:/.test(d[1]) ? d : [d[0], path.join(
    'file:cfpackage', path.relative(root, path.resolve(d[1].substr(5))))];
};

// Adjust local dependencies to our packed app structure and write new
// package.json
var repackage = function(root, cb) {
  var mod = require(path.join(process.cwd(), 'package.json'));
  var loc = partial(local, root);
  var rmod = extend({}, mod, {
    dependencies: object(map(pairs(mod.dependencies), loc))
  }, {
    devDependencies: object(map(pairs(mod.devDependencies), loc))
  });
  fs.writeFile(path.join('cfpackage', 'package.json'),
    JSON.stringify(rmod, undefined, 2), cb);
};

// Produce the packaged app zip
var zip = function(ignore, cb) {
  fs.unlink(path.resolve('cfpackage', 'app.zip'), function(err) {
    if(err) noop();

    // We're using the system zip command here, may be better to use a
    // Javascript zip library instead
    var files = '-type f -regex "./lib.*" -or -regex "./config.*" -or -regex "./package.json.*"';
    var ex = cp.exec('(find . cfpackage/lib/* ' + files + ' | zip -q -x@' +
      ignore + ' -@ cfpackage/app.zip) && ' +
      '(zip -q -j cfpackage/app.zip cfpackage/package.json)', {
        cwd: process.cwd()
      });


    ex.stdout.on('data', function(data) {
      process.stdout.write(data);
    });
    ex.stderr.on('data', function(data) {
      process.stderr.write(data);
    });
    ex.on('close', function(code) {
      cb(code);
    });
  });
};

// Return the Abacus root directory
var rootDir = function(dir) {
  if(dir === '/') return dir;
  try {
    if(JSON.parse(fs.readFileSync(
      path.resolve(dir, 'package.json')).toString()).name === 'auto-scaler-service-broker') {
      return dir;
    }
    return rootDir(path.resolve(dir, '..'));
  }
  catch (e) {
    return rootDir(path.resolve(dir, '..'));
  }
};

// Package an app for deployment to Cloud Foundry
var runCLI = function() {
  // Parse command line options
  commander
    // Accept root directory of local dependencies as a parameter, default
    // to the Abacus root directory
    .option(
      '-r, --root <dir>', 'root local dependencies directory',
      rootDir(process.cwd()))
    .parse(process.argv);

  // Create the directories we need
  mkdirs(commander.root, function(err) {
    if(err) {
      console.log('Couldn\'t setup cfpack layout -', err);
      process.exit(1);
    }

    // Generate the repackaged package.json
    repackage(commander.root, function(err) {
      if(err) {
        console.log('Couldn\'t write package.json -', err);
        process.exit(1);
      }

      // Produce the packaged app zip
      zip(path.join(commander.root, '.gitignore'), function(err) {
        if(err) {
          console.log('Couldn\'t produce cfpackage/app.zip -', err);
          process.exit(1);
        }
      });
    });
  });
};

// Export our CLI
module.exports.runCLI = runCLI;

