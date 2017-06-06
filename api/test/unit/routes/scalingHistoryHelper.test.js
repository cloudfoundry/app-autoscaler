"use strict"
var path = require("path");
var expect = require("chai").expect;
var helper = require(path.join(__dirname, "../../../lib/routes/scalingHistoryHelper.js"));
var defaultRequest = function() {
  var obj = {
    params: {
      "guid": "123456"
    },
    query: {
      "start-time": "100",
      "end-time": "200",
      "order": "desc",
      "page": "1",
      "results-per-page": "10"
    }
  }
  return obj;
}
describe("ScalingHistoryHelper", function() {

  describe("validate parameters", function() {
    var validateResult;
    var requestObj;
    beforeEach(function() {
      requestObj = defaultRequest();
    });
    context("all parameters are valid", function() {
      it("valid is true", function() {
        validateResult = helper.validateParameter(requestObj);
        expect(validateResult.valid).to.equal(true);
      });
    });

    context("validate guid", function() {
      context("guid is undefined", function() {
        it("valid is false", function() {
          delete requestObj.params.guid;
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("app guid is required");
        });
      });

      context("guid is null", function() {
        it("valid is false", function() {
          requestObj.params.guid = null;
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("app guid is required");
        });
      });

      context("guid is empty", function() {
        it("valid is false", function() {
          requestObj.params.guid = "";
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("app guid is required");
        });
      });
    });

    context("validate start-time", function() {
      context("start-time is undefined", function() {
        it("valid is false", function() {
          delete requestObj.query["start-time"];

          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("start-time is required");
        });
      });

      context("start-time is null", function() {
        it("valid is false", function() {
          requestObj.query["start-time"] = null;
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("start-time is required");
        });
      });

      context("start-time is not integer", function() {
        it("valid is false", function() {
          requestObj.query["start-time"] = "not-integer";
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("start-time must be an integer");
        });
      });
    });

    context("validate end-time", function() {
      context("end-time is undefined", function() {
        it("valid is false", function() {
          delete requestObj.query["end-time"];
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("end-time is required");
        });
      });

      context("end-time is null", function() {
        it("valid is false", function() {
          requestObj.query["end-time"] = null;
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("end-time is required");
        });
      });

      context("end-time is not integer", function() {
        it("valid is false", function() {
          requestObj.query["end-time"] = "not-integer";
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("end-time must be an integer");
        });
      });
    });

    context("validate start-time and end-time", function() {
      context("start-time is greater than end-time", function() {
        it("valid false", function() {
          requestObj.query["start-time"] = 100;
          requestObj.query["end-time"] = 0;
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("start-time must be smaller than end-time");
        });

      });
    });

    context("validate page", function() {
      context("page is undefined", function() {
        it("valid is false", function() {
          delete requestObj.query["page"];
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("page is required");
        });
      });

      context("page is null", function() {
        it("valid is false", function() {
          requestObj.query["page"] = null;
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("page is required");
        });
      });

      context("page is not integer", function() {
        it("valid is false", function() {
          requestObj.query["page"] = "not-integer";
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("page must be an integer");
        });
      });

      context("page is an integer but smaller than 0", function() {
        it("valid is false", function() {
          requestObj.query["page"] = -1;
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("page must be greater or equal than 0");
        });
      });
    });

    context("validate results-per-page", function() {
      context("results-per-page is undefined", function() {
        it("valid is false", function() {
          delete requestObj.query["results-per-page"];
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("results-per-page is required");
        });
      });

      context("results-per-page is null", function() {
        it("valid is false", function() {
          requestObj.query["results-per-page"] = null;
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("results-per-page is required");
        });
      });

      context("results-per-page is not integer", function() {
        it("valid is false", function() {
          requestObj.query["results-per-page"] = "not-integer";
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("results-per-page must be an integer");
        });
      });

      context("results-per-page is an integer but smaller than 0", function() {
        it("valid is false", function() {
          requestObj.query["results-per-page"] = -1;
          validateResult = helper.validateParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("results-per-page must be greater or equal than 0");
        });
      });
    });

  });
  describe("pagination", function() {
    var list = [];
    var result = {};
    context("paginate the list", function() {
      list = [
        { 'app_guid': "theAppId", 'timestamp': 300, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_guid': "theAppId", 'timestamp': 250, 'scaling_type': 1, 'status': 1, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_guid': "theAppId", 'timestamp': 200, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_guid': "theAppId", 'timestamp': 150, 'scaling_type': 1, 'status': 1, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_guid': "theAppId", 'timestamp': 100, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' }
      ];
      it("get the 1st page", function() {
        result = helper.pagination(list, 0, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 0,
          resources: list.slice(0, 2)
        });
      });

      it("get the 2nd page", function() {
        result = helper.pagination(list, 1, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 1,
          resources: list.slice(2, 4)
        });
      });

      it("get the 3rd page and only has one record", function() {
        result = helper.pagination(list, 2, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 2,
          resources: list.slice(4)
        });
      });

      it("get the 4th page and there is no record", function() {
        result = helper.pagination(list, 3, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 3,
          resources: []
        });
      });

    });
  });
});
