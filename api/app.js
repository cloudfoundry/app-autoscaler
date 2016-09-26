var express = require('express');
var bodyParser = require('body-parser');
var logger = require('./lib/log/logger');
var HttpStatus = require('http-status-codes');
var bootstrap = require('./lib/utils/bootstrap');
var app = express();
app.use(bodyParser.json());
app.use(bodyParser.urlencoded({ extended: false }));

bootstrap.boot(function(error){
  if (!error) {
    var server = app.listen(process.env.PORT || 3002, function() {
      logger.info('Autoscaler API server started',{'port':server.address().port} ); 
      var policies = require('./lib/routes/policies');
      app.use('/v1/policies',policies);
      app.use(function(err, req, res, next) {
        var errorResponse = {};
        if (err) {
          errorResponse = {
            'success': false,
            'error': err,
            'result': null
          };
        }
        res.status(HttpStatus.BAD_REQUEST).json(errorResponse);
      });
      
    });  
  } else {
    logger.error ('Autoscaler API server failed to start ',{'error': error} );
  }
});
module.exports = app; 