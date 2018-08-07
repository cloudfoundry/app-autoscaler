"use strict"
var path = require("path");
var expect = require("chai").expect;
var helper = require(path.join(__dirname, "../../../lib/routes/metricHelper.js"));
var defaultRequest = function() {
  var obj = {
    params: {
      "app_id": "123456",
      "metric_type": "memoryused"
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
describe("metricHelper", function() {

  describe("validate parameters", function() {
    var validateResult;
    var requestObj;
    beforeEach(function() {
      requestObj = defaultRequest();
    });
    context("validate metric_type", function() {
      context("metric_type is undefined", function() {
        it("return false", function() {
          delete requestObj.params.metric_type;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("metric_type is required");
        });
      });

      context("metric_type is null", function() {
        it("return false", function() {
          requestObj.params.metric_type = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("metric_type is required");
        });
      });

      context("metric_type is empty", function() {
        it("return false", function() {
          requestObj.params.metric_type = "";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("metric_type is required");
        });
      });
    });

  });
});