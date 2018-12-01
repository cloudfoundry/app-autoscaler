'use strict'

var path = require("path");
var expect = require("chai").expect;
var helper = require(path.join(__dirname, "../../../lib/routes/routeHelper.js"));

describe("routeHelper", function() {
  describe("isMissing", function() {
    var toCheckObj;
    beforeEach(function() {
      toCheckObj = { value: "anything" };
    });
    context("check valid value", function() {
      it("return false", function() {
        expect(helper.isMissing(toCheckObj.value)).to.equal(false);
      });
    });
    context("check null", function() {
      it("return true", function() {
        toCheckObj.value = null;
        expect(helper.isMissing(toCheckObj.value)).to.equal(true);
      });
    });
    context("check undefined", function() {
      it("return true", function() {
        delete toCheckObj.value;
        expect(helper.isMissing(toCheckObj.value)).to.equal(true);
      });
    });
  });

  describe("isInteger", function() {
    var intVar;
    beforeEach(function() {
      intVar = 1;
    });
    context("check integer", function() {
      it("return true", function() {
        expect(helper.isInteger(intVar)).to.equal(true);
      });
    });
    context("check string", function() {
      it("return false", function() {
        intVar = "not-integer-string";
        expect(helper.isInteger(intVar)).to.equal(false);
      });
    });
    context("check array", function() {
      it("return false", function() {
        intVar = [];
        expect(helper.isInteger(intVar)).to.equal(false);
      });
    });
    context("check obj", function() {
      it("return false", function() {
        intVar = {};
        expect(helper.isInteger(intVar)).to.equal(false);
      });
    });
  });

  describe("isString", function() {
    var strVar;
    beforeEach(function() {
      strVar = "string";
    });
    context("check string", function() {
      it("return true", function() {
        expect(helper.isString(strVar)).to.equal(true);
      });
    });
    context("check number", function() {
      it("return false", function() {
        strVar = 123;
        expect(helper.isString(strVar)).to.equal(false);
      });
    });
    context("check array", function() {
      it("return false", function() {
        strVar = [];
        expect(helper.isInteger(strVar)).to.equal(false);
      });
    });
    context("check obj", function() {
      it("return false", function() {
        strVar = {};
        expect(helper.isInteger(strVar)).to.equal(false);
      });
    });
  });

  describe("parseParameter", function() {
    var validateResult;
    var requestObj;
    var defaultRequest = function() {
      var obj = {
        params: {
          "app_id": "123456"
        },
        query: {
          "start-time": "100",
          "end-time": "200",
          "order-direction": "DESC",
          "order": "DESC",
          "page": "1",
          "results-per-page": "10"
        }
      }
      return obj;
    }
    beforeEach(function() {
      requestObj = defaultRequest();
    });
    context("all parameters are valid", function() {
      it("return true", function() {
        validateResult = helper.parseParameter(requestObj);
        expect(validateResult.valid).to.equal(true);
      });
    });

    context("validate app_id", function() {
      context("app_id is undefined", function() {
        it("return false", function() {
          delete requestObj.params.app_id;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("app_id is required");
        });
      });

      context("app_id is null", function() {
        it("return false", function() {
          requestObj.params.app_id = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("app_id is required");
        });
      });

      context("app_id is empty", function() {
        it("return false", function() {
          requestObj.params.app_id = "";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("app_id is required");
        });
      });
    });

    context("validate start-time", function() {
      context("start-time is undefined", function() {
        it("return true, startTime is set to 0", function() {
          delete requestObj.query["start-time"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.startTime).to.equal(0);
        });
      });

      context("start-time is null", function() {
        it("return true, startTime is set to 0", function() {
          requestObj.query["start-time"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.startTime).to.equal(0);
        });
      });

      context("start-time is not integer", function() {
        it("return false", function() {
          requestObj.query["start-time"] = "not-integer";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("start-time must be an integer");
        });
      });
    });

    context("validate end-time", function() {
      context("end-time is undefined", function() {
        it("return true, endTime is set to -1", function() {
          delete requestObj.query["end-time"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.endTime).to.equal(-1);
        });
      });

      context("end-time is null", function() {
        it("return true, endTime is set to -1", function() {
          requestObj.query["end-time"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.endTime).to.equal(-1);
        });
      });

      context("end-time is not integer", function() {
        it("return false", function() {
          requestObj.query["end-time"] = "not-integer";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("end-time must be an integer");
        });
      });
    });
    context("validate order-direction/order", function() {
      context("both order-direction and order are valid", function() {
        it("return true, order-direction take precedence", function() {
          requestObj.query["order-direction"] = "ASC";
          requestObj.query["order"] = "DESC";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters["order"]).to.equal("ASC");
        });
      });
      context("both order-direction and order are undefined", function() {
        it("return true, order is set to ASC", function() {
          delete requestObj.query["order-direction"];
          delete requestObj.query["order"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters["order"]).to.equal("DESC");
        });
      });
      context("both order-direction and order are null", function() {
        it("return true, order is set to ASC", function() {
          requestObj.query["order-direction"] = null;
          requestObj.query["order"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters["order"]).to.equal("DESC");
        });
      });
      context("order-direction is not undefined or null", function() {
        context("order-direction is not string", function() {
          it("return false", function() {
            requestObj.query["order-direction"] = 1;
            validateResult = helper.parseParameter(requestObj);
            expect(validateResult.valid).to.equal(false);
            expect(validateResult.message).to.equal("order-direction must be a string");
          });

        });
        context("order-direction is not DESC or ASC", function() {
          it("return false", function() {
            requestObj.query["order-direction"] = "not-DESC-ASC";
            validateResult = helper.parseParameter(requestObj);
            expect(validateResult.valid).to.equal(false);
            expect(validateResult.message).to.equal("order-direction must be DESC or ASC");
          });

        });
      });

      context("order-direction is undefined and order has value", function() {
        context("order is not string", function() {
          it("return false", function() {
            delete requestObj.query["order-direction"]
            requestObj.query["order"] = 1;
            validateResult = helper.parseParameter(requestObj);
            expect(validateResult.valid).to.equal(false);
            expect(validateResult.message).to.equal("order-direction must be a string");
          });

        });
        context("order is not DESC or ASC", function() {
          it("return false", function() {
            delete requestObj.query["order-direction"]
            requestObj.query["order"] = "not-DESC-ASC";
            validateResult = helper.parseParameter(requestObj);
            expect(validateResult.valid).to.equal(false);
            expect(validateResult.message).to.equal("order-direction must be DESC or ASC");
          });
        });
      });

      context("order-direction is null and order has value", function() {
        context("order is not string", function() {
          it("return false", function() {
            requestObj.query["order-direction"] = null;
            requestObj.query["order"] = 1;
            validateResult = helper.parseParameter(requestObj);
            expect(validateResult.valid).to.equal(false);
            expect(validateResult.message).to.equal("order-direction must be a string");
          });

        });
        context("order is not DESC or ASC", function() {
          it("return false", function() {
            requestObj.query["order-direction"] = null;
            requestObj.query["order"] = "not-DESC-ASC";
            validateResult = helper.parseParameter(requestObj);
            expect(validateResult.valid).to.equal(false);
            expect(validateResult.message).to.equal("order-direction must be DESC or ASC");
          });

        });
      });
    });

    context("validate page", function() {
      context("page is undefined", function() {
        it("return true, page is set to 1", function() {
          delete requestObj.query["page"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.page).to.equal(1);
        });
      });

      context("page is null", function() {
        it("return true, page is set to 1", function() {
          requestObj.query["page"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.page).to.equal(1);
        });
      });

      context("page is not integer", function() {
        it("return false", function() {
          requestObj.query["page"] = "not-integer";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("page must be an integer");
        });
      });

      context("page is an integer but smaller than 1", function() {
        it("return false", function() {
          requestObj.query["page"] = 0;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("page must be greater than 0");
        });
      });
    });

    context("validate results-per-page", function() {
      context("results-per-page is undefined", function() {
        it("return true, reulstsPerPage is set to 50", function() {
          delete requestObj.query["results-per-page"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.resultsPerPage).to.equal(50);
        });
      });

      context("results-per-page is null", function() {
        it("return true, reulstsPerPage is set to 50", function() {
          requestObj.query["results-per-page"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.resultsPerPage).to.equal(50);
        });
      });

      context("results-per-page is not integer", function() {
        it("return false", function() {
          requestObj.query["results-per-page"] = "not-integer";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("results-per-page must be an integer");
        });
      });

      context("results-per-page is an integer but smaller than 1", function() {
        it("return false", function() {
          requestObj.query["results-per-page"] = 0;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("results-per-page must be greater than 0");
        });
      });
    });
  });

  describe("pagination", function() {
    var resultList = [];
    var result = {};
    var request = {};
    context("paginate the resultList", function() {
      resultList = [
        { 'app_id': "theAppId", 'timestamp': 300, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 250, 'scaling_type': 1, 'status': 1, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 200, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 150, 'scaling_type': 1, 'status': 1, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 100, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' }
      ];
      context("get the 1st page", function() {
        beforeEach(function() {
          request = {
            "baseUrl": "/v1",
            "path": "/someroute",
            "query": {
              "page": 1,
              "resuts-per-page": 2
            }
          };
        });
        it("return the 1st page", function() {
          result = helper.pagination(resultList, 1, 2, request);
          expect(result).to.deep.equal({
            total_results: 5,
            total_pages: 3,
            page: 1,
            prev_url: null,
            next_url: "/v1/someroute?page=2&resuts-per-page=2",
            resources: resultList.slice(0, 2)
          });
        });
      });

      context("get the 2nd page", function() {
        beforeEach(function() {
          request = {
            "baseUrl": "/v1",
            "path": "/someroute",
            "query": {
              "page": 2,
              "resuts-per-page": 2
            }
          };
        });
        it("get the 2nd page", function() {
          result = helper.pagination(resultList, 2, 2, request);
          expect(result).to.deep.equal({
            total_results: 5,
            total_pages: 3,
            page: 2,
            prev_url: "/v1/someroute?page=1&resuts-per-page=2",
            next_url: "/v1/someroute?page=3&resuts-per-page=2",
            resources: resultList.slice(2, 4)
          });
        });
      });

      context("get the 3rd page", function() {
        beforeEach(function() {
          request = {
            "baseUrl": "/v1",
            "path": "/someroute",
            "query": {
              "page": 3,
              "resuts-per-page": 2
            }
          };
        });
        it("get the 3rd page and only has one record", function() {
          result = helper.pagination(resultList, 3, 2, request);
          expect(result).to.deep.equal({
            total_results: 5,
            total_pages: 3,
            page: 3,
            prev_url: "/v1/someroute?page=2&resuts-per-page=2",
            next_url: null,
            resources: resultList.slice(4)
          });
        });
      });

      context("get the 4th page", function() {
        beforeEach(function() {
          request = {
            "baseUrl": "/v1",
            "path": "/someroute",
            "query": {
              "page": 4,
              "resuts-per-page": 2
            }
          };
        });
        it("get the 4th page and there is no record", function() {
          result = helper.pagination(resultList, 4, 2, request);
          expect(result).to.deep.equal({
            total_results: 5,
            total_pages: 3,
            page: 4,
            prev_url: "/v1/someroute?page=3&resuts-per-page=2",
            next_url: null,
            resources: []
          });
        });
      });

      context("get the 5th page", function() {
        beforeEach(function() {
          request = {
            "baseUrl": "/v1",
            "path": "/someroute",
            "query": {
              "page": 5,
              "resuts-per-page": 2
            }
          };

        });
        it("get the 5th page and there is no record and the prev_url and next_url are both null", function() {
          result = helper.pagination(resultList, 5, 2, request);
          expect(result).to.deep.equal({
            total_results: 5,
            total_pages: 3,
            page: 5,
            prev_url: null,
            next_url: null,
            resources: []
          });
        });
      });


    });
  });

  describe("getPageUrl", function() {
    var request;
    beforeEach(function() {
      request = {
        "baseUrl": "/v1",
        "path": "/someroute",
        "query": {
          "page": 1,
          "resuts-per-page": 2
        }
      };
    });
    context("request.query is null", function() {
      beforeEach(function() {
        request.query = null;
      });
      it("return null",function(){
        expect(helper.getPageUrl(request,2)).to.equal(null);
      });
    });
    context("request.query is undefined", function() {
      beforeEach(function() {
        delete request.query;
      });
      it("return null",function(){
        expect(helper.getPageUrl(request,2)).to.equal(null);
      });
    });
    context("get page url", function() {
      it("return page url",function(){
        expect(helper.getPageUrl(request,2)).to.equal("/v1/someroute?page=2&resuts-per-page=2");
      });
    });
  })
});