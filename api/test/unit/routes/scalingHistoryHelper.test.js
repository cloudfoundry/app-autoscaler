"use strict"
var path = require("path");
var expect = require("chai").expect;
var helper = require(path.join(__dirname, "../../../lib/routes/scalingHistoryHelper.js"));
var defaultRequest = function() {
  var obj = {
    params: {
      "app_id": "123456"
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
        validateResult = helper.parseParameter(requestObj);
        expect(validateResult.valid).to.equal(true);
      });
    });

    context("validate app_id", function() {
      context("app_id is undefined", function() {
        it("valid is false", function() {
          delete requestObj.params.app_id;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("app_id is required");
        });
      });

      context("app_id is null", function() {
        it("valid is false", function() {
          requestObj.params.app_id = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("app_id is required");
        });
      });

      context("app_id is empty", function() {
        it("valid is false", function() {
          requestObj.params.app_id = "";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("app_id is required");
        });
      });
    });

    context("validate start-time", function() {
      context("start-time is undefined", function() {
        it("valid is true, startTime is set to 0", function() {
          delete requestObj.query["start-time"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.startTime).to.equal(0);
        });
      });

      context("start-time is null", function() {
        it("valid is true, startTime is set to 0", function() {
          requestObj.query["start-time"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.startTime).to.equal(0);
        });
      });

      context("start-time is not integer", function() {
        it("valid is false", function() {
          requestObj.query["start-time"] = "not-integer";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("start-time must be an integer");
        });
      });
    });

    context("validate end-time", function() {
      context("end-time is undefined", function() {
        it("valid is true, endTime is set to -1", function() {
          delete requestObj.query["end-time"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.endTime).to.equal(-1);
        });
      });

      context("end-time is null", function() {
        it("valid is true, endTime is set to -1", function() {
          requestObj.query["end-time"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.endTime).to.equal(-1);
        });
      });

      context("end-time is not integer", function() {
        it("valid is false", function() {
          requestObj.query["end-time"] = "not-integer";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("end-time must be an integer");
        });
      });
    });
    context("validate order", function() {
      context("order is undefined", function() {
        it("valid is true, order is set to desc", function() {
          delete requestObj.query["order"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.order).to.equal("desc");
        });

      });
      context("order is null", function() {
        it("valid is true, order is set to desc", function() {
          requestObj.query["order"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.order).to.equal("desc");
        });

      });
      context("order is not string", function() {
        it("valid is false", function() {
          requestObj.query["order"] = 1;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("order must be a string");
        });

      });
      context("order is not desc or asc", function() {
        it("valid is false", function() {
          requestObj.query["order"] = "not-desc-asc";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("order must be desc or asc");
        });

      });
    });
    context("validate page", function() {
      context("page is undefined", function() {
        it("valid is true, page is set to 1", function() {
          delete requestObj.query["page"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.page).to.equal(1);
        });
      });

      context("page is null", function() {
        it("valid is true, page is set to 1", function() {
          requestObj.query["page"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.page).to.equal(1);
        });
      });

      context("page is not integer", function() {
        it("valid is false", function() {
          requestObj.query["page"] = "not-integer";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("page must be an integer");
        });
      });

      context("page is an integer but smaller than 1", function() {
        it("valid is false", function() {
          requestObj.query["page"] = 0;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("page must be greater than 0");
        });
      });
    });

    context("validate results-per-page", function() {
      context("results-per-page is undefined", function() {
        it("valid is true, reulstsPerPage is set to 10", function() {
          delete requestObj.query["results-per-page"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.resultsPerPage).to.equal(10);
        });
      });

      context("results-per-page is null", function() {
        it("valid is true, reulstsPerPage is set to 10", function() {
          requestObj.query["results-per-page"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters.resultsPerPage).to.equal(10);
        });
      });

      context("results-per-page is not integer", function() {
        it("valid is false", function() {
          requestObj.query["results-per-page"] = "not-integer";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("results-per-page must be an integer");
        });
      });

      context("results-per-page is an integer but smaller than 1", function() {
        it("valid is false", function() {
          requestObj.query["results-per-page"] = 0;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("results-per-page must be greater than 0");
        });
      });
    });

  });
  describe("pagination", function() {
    var scalingHistoryList = [];
    var result = {};
    context("paginate the scalingHistoryList", function() {
      scalingHistoryList = [
        { 'app_id': "theAppId", 'timestamp': 300, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 250, 'scaling_type': 1, 'status': 1, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 200, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 150, 'scaling_type': 1, 'status': 1, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 100, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' }
      ];
      it("get the 1st page", function() {
        result = helper.pagination(scalingHistoryList, 1, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 1,
          resources: scalingHistoryList.slice(0, 2)
        });
      });

      it("get the 2nd page", function() {
        result = helper.pagination(scalingHistoryList, 2, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 2,
          resources: scalingHistoryList.slice(2, 4)
        });
      });

      it("get the 3rd page and only has one record", function() {
        result = helper.pagination(scalingHistoryList, 3, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 3,
          resources: scalingHistoryList.slice(4)
        });
      });

      it("get the 4th page and there is no record", function() {
        result = helper.pagination(scalingHistoryList, 4, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 4,
          resources: []
        });
      });

    });
  });
});
