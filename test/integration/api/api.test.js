var request=require('request');
var expect=require('chai').expect;
var fs=require('fs');
var async=require('async');
var path = require('path');
var schedulerURI=process.env.SCHEDULER_URI;
var apiServerURI=process.env.APISERVER_URI;
var spawn = require('child_process').spawn;

describe('API Server-Scheduler Integration tests', function(){
	var scheduler;
	var apiServer;
	
	// Health Check for Scheduler 
	var startScheduler = function(cb){
		process.stdout.write('Starting the Scheduler...');
		scheduler = spawn ('java', ['-jar', path.join(__dirname,'../../../scheduler/target/scheduler-1.0-SNAPSHOT.war')]);
		var timesRun = 0;
		var schedulerIntervalId = setInterval(function(){
			request({url:schedulerURI+'/health',method:'GET',json:true},function(error,response,body){
				timesRun++;
				process.stdout.write ('...');
				if (timesRun >= 10){
					clearInterval(schedulerIntervalId);
					return cb(new Error('Scheduler health check timed out'));
				}
				if (response && response.statusCode === 200){
					console.log('\nScheduler Health check succeeded. Server is up and running !');
					clearInterval(schedulerIntervalId);
					return cb(null);
				}
			});
		}, 2000);
	};
	
	// Health Check for API Server 
	var startApiServer = function(cb){
		process.stdout.write('Starting the API Server...');
		apiServer = spawn ('node', [path.join(__dirname,'../../../api/app.js')]);
		var timesRun = 0;
		var apiIntervalId = setInterval(function(){
			request({url:apiServerURI+'/health',method:'GET',json:true},function(error,response,body){
			timesRun++;
			process.stdout.write ('...');
			if (timesRun >= 10){
				clearInterval(apiIntervalId);
				return cb(new Error('API Server health check timed out'));
			}
			if (response && response.statusCode === 200){
				console.log('\nAPI Server Health check succeeded. Server is up and running !');
				clearInterval(apiIntervalId);
				return cb(null);
			}});
		}, 2000);
	};

	before(function(done){ 
		fakePolicy=JSON.parse(fs.readFileSync(__dirname+'/fakePolicy.json','utf8'));
		async.series([startScheduler,startApiServer], function(err){
			done(err);
		});
	});
	
    after(function(done) {
        async.series([
                      function(callback) {
                          try {
                              scheduler.kill();
                          } catch (e) {
                              console.error ('Error while trying to stop the Scheduler. Attempting to stop the API server', e);
                              // Not passing the error to the callback but in the results 
                              // to ensure that the next function in the series to stop the apiserver is called
                              callback(null, e);
						  }
						  console.log ('Scheduler stopped successfully')
                          callback();
                      },
                      function(callback) {
                          try {
                              apiServer.kill();
                          } catch (e) {
                              console.error ('Error while trying to stop the API Server.', e);
                              callback(null, e);
                          }
                          console.log ('API Server stopped successfully')
                          callback();
                      }
                  ],
                  function(err, results) {
                      if (results[0] || results[1])
                          console.error ('Some error may have occurred while trying to stop the API Server or the Scheduler');
                      done ();
                  }
        );
    });
    
	context('Create Policy with schedules ',function(){
		// Cleanup the DB before running every context
		beforeEach(function(done){
			var deletePolicyOptions={
					url:apiServerURI+'/v1/policies/dummy',
					method:'DELETE',
					json:true
			};
			request(deletePolicyOptions,function(error,response,body){
				done();
			});
		});

		it('Should create a policy and associated schedules',function(done){
			async.series([	
			function createDummyPolicy(callback){
				var createOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'PUT',
						body:fakePolicy,
						json:true
				};
				request(createOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(201);
					expect(response.body.success).to.equal(true);
					expect(response.body.error).to.be.null;
					expect(response.body.result.policy_json).eql(fakePolicy);
					expect(response.body.result.app_id).to.equal('dummy');
					callback(error,response);
				});
			},
			function getDummyPolicy(callback){
				var policyOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'GET',
						json:true
				};
				request(policyOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(200);
					expect(response.body).to.deep.equal(fakePolicy);
					callback(error,response);
				});	
			},
			function getDummySchedule(callback){
				var getSchedulesOptions={
						url:schedulerURI+'/v2/schedules/dummy',
						method:'GET',
						json:true
				};
				request(getSchedulesOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(200);
					expect(response.body.schedules.recurring_schedule).to.have.lengthOf(4);
					expect(response.body.schedules.specific_date).to.have.lengthOf(2);
					callback(error,response);
				});
			}
			],done);
		});
		
		it('Should fail to create policy and associated schedules due to some validation error with the policy',function(done){
			var newFakePolicy=JSON.parse(JSON.stringify(fakePolicy));
			newFakePolicy.instance_min_count=5;
			async.series([
			function createDummyPolicy(callback){
				var createOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'PUT',
						body:newFakePolicy,
						json:true
				};
				request(createOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(400);
					callback(error,response);
				});
			},
			function getDummyPolicy(callback){
				var policyOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'GET',
						json:true
				};
				request(policyOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(404);
					callback(error,response);
				});
			},
			function getDummySchedule(callback){
				var getSchedulesOptions={
						url:schedulerURI+'/v2/schedules/dummy',
						method:'GET',
						json:true
				};
				request(getSchedulesOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(404);
					callback(error,response);
				});
			}
			],done);
		});
	});
	
	context(' Create policy without schedules ',function(){
		// Cleanup the DB before running every context
		beforeEach(function(done){
			var deletePolicyOptions={
					url:apiServerURI+'/v1/policies/dummy',
					method:'DELETE',
					json:true
			};
			request(deletePolicyOptions,function(error,response,body){
				done();
			});		
		});	

		it('should create the policy only',function(done){
			var policyWithoutSchedules=JSON.parse(JSON.stringify(fakePolicy));
			delete policyWithoutSchedules.schedules;
			async.series([	
			function createDummyPolicy(callback){
				var createOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'PUT',
						body:policyWithoutSchedules,
						json:true
				};
				request(createOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(201);
					expect(response.body.success).to.equal(true);
					expect(response.body.error).to.be.null;
					expect(response.body.result.app_id).to.equal('dummy');
					callback(error,response);
				});
			},
			function getDummyPolicy(callback){
				var policyOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'GET',
						json:true
				};
				request(policyOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(200);
					callback(error,response);
				});	
			},
			function getDummySchedule(callback){
				var getSchedulesOptions={
						url:schedulerURI+'/v2/schedules/dummy',
						method:'GET',
						json:true
				};
				request(getSchedulesOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(404);
					callback(error,response);
				});
			}
			],done);
		});
		
	});
	
	context('Update policy with schedules',function(){
		before(function(done){
			var deletePolicyOptions={
					url:apiServerURI+'/v1/policies/dummy',
					method:'DELETE',
					json:true
			};
			request(deletePolicyOptions,function(error,response,body){
				
				var createOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'PUT',
						body:fakePolicy,
						json:true
				};
				request(createOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(201);
					expect(response.body.success).to.equal(true);
					expect(response.body.error).to.be.null;
					expect(response.body.result.policy_json).eql(fakePolicy);
					expect(response.body.result.app_id).to.equal('dummy');
					done();
				});
			});		
		});

		it('Should update the policy and associated schedules',function(done){
			var newFakePolicy=JSON.parse(JSON.stringify(fakePolicy));
			newFakePolicy.schedules.recurring_schedule[0].instance_max_count=8;
			newFakePolicy.instance_min_count=2;
			async.series([	
			function updateDummyPolicy(callback){
				var updateOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'PUT',
						body:newFakePolicy,
						json:true
				};
				request(updateOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(200);
					expect(response.body.success).to.equal(true);
					expect(response.body.error).to.be.null;
					expect(response.body.result[0].policy_json).eql(newFakePolicy);
					expect(response.body.result[0].app_id).to.equal('dummy');
					callback(error,response);
				});
			},
			function getDummyPolicy(callback){
				var policyOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'GET',
						json:true
				};
				request(policyOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(200);
					expect(response.body).to.deep.equal(newFakePolicy);
					callback(error,response);
				});	
			},
			function getDummySchedules(callback){
				var getSchedulesOptions={
						url:schedulerURI+'/v2/schedules/dummy',
						method:'GET',
						json:true
				};
				request(getSchedulesOptions,function(error,response,body){
					expect(error).to.be.null;
					expect(response.statusCode).to.equal(200);
					expect(response.body.schedules.recurring_schedule).to.have.lengthOf(4);
					expect(response.body.schedules.specific_date).to.have.lengthOf(2);
					callback(error,response);
				});
			}
			],done);
		});

	});
		
	context('Delete Policy with schedules ',function(){
		context('for a non-existing app', function () {
			before(function(done){
				var deletePolicyOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'DELETE',
						json:true
				};
				request(deletePolicyOptions,function(error,response,body){
					done();
				});
			});	

			it('Should return a 404 status code',function(done){
					var deletePolicyOptions={
							url:apiServerURI+'/v1/policies/dummy',
							method:'DELETE',
							json:true
					};
					request(deletePolicyOptions,function(error,response,body){
						expect(response.statusCode).to.equal(404);
						expect(response.body.success).to.equal(false);
						expect(response.body.error.statusCode).to.equal(404);
						done();
					});
			});
		});

		context('for an existing app', function() {
			before(function(done){
				var deletePolicyOptions={
						url:apiServerURI+'/v1/policies/dummy',
						method:'DELETE',
						json:true
				};
				request(deletePolicyOptions,function(error,response,body){
					var createOptions={
							url:apiServerURI+'/v1/policies/dummy',
							method:'PUT',
							body:fakePolicy,
							json:true
					};
					request(createOptions,function(error,response,body){
						expect(error).to.be.null;
						expect(response.statusCode).to.equal(201);
						expect(response.body.success).to.equal(true);
						expect(response.body.error).to.be.null;
						expect(response.body.result.policy_json).eql(fakePolicy);
						expect(response.body.result.app_id).to.equal('dummy');
						done();
					});
				});
			});

			it('Should delete the existing policy and associated schedules',function(done){
				async.series([
				function getDummyPolicy(callback){
					var policyOptions={
							url:apiServerURI+'/v1/policies/dummy',
							method:'GET',
							json:true
					};
					request(policyOptions,function(error,response,body){
						expect(error).to.be.null;
						expect(response.statusCode).to.equal(200);
						expect(response.body).to.deep.equal(fakePolicy);
						callback(error,response);
					});	
				},
				function getDummySchedule(callback){
					var getSchedulesOptions={
							url:schedulerURI+'/v2/schedules/dummy',
							method:'GET',
							json:true
					};
					request(getSchedulesOptions,function(error,response,body){
						expect(error).to.be.null;
						expect(response.statusCode).to.equal(200);
						expect(response.body.schedules.recurring_schedule).to.have.lengthOf(4);
						expect(response.body.schedules.specific_date).to.have.lengthOf(2);
						callback(error,response);
					});
				},
				function deletePolicy(callback){
					var deletePolicyOptions={
							url:apiServerURI+'/v1/policies/dummy',
							method:'DELETE',
							json:true
					};
					request(deletePolicyOptions,function(error,response,body){
						expect(error).to.be.null;
						expect(response.statusCode).to.equal(200);
						callback(error,response);
					});
				},
				function getDummyPolicy(callback){
					var policyOptions={
							url:apiServerURI+'/v1/policies/dummy',
							method:'GET',
							json:true
					};
					request(policyOptions,function(error,response,body){
						expect(error).to.be.null;
						expect(response.statusCode).to.equal(404);
						callback(error,response);
					});
				},
				function getDummySchedule(callback){
					var getSchedulesOptions={
							url:schedulerURI+'/v2/schedules/dummy',
							method:'GET',
							json:true
					};
					request(getSchedulesOptions,function(error,response,body){
						expect(error).to.be.null;
						expect(response.statusCode).to.equal(404);
						callback(error,response);
					});
				},
				],done);
			});
		});
		
	});
});
