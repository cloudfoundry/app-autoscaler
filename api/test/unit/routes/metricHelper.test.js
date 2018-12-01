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
      "instance-index": 1,
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
    context("all parameters and query are valid", function() {
      it("return true", function() {
        validateResult = helper.parseParameter(requestObj);
        expect(validateResult.valid).to.equal(true);
        expect(validateResult.parameters).to.deep.equal({
          "appId": requestObj.params.app_id,
          "metricType": requestObj.params.metric_type,
          "instanceIndex": parseInt(requestObj.query["instance-index"]),
          "startTime": parseInt(requestObj.query["start-time"]),
          "endTime": parseInt(requestObj.query["end-time"]),
          "order": requestObj.query["order-direction"],
          "page": parseInt(requestObj.query["page"]),
          "resultsPerPage": parseInt(requestObj.query["results-per-page"])
        });
      });
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

    context("validate instance-index", function() {
      context("instance-index is undefined", function() {
        it("return true", function() {
          delete requestObj.query["instance-index"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
        });
      });

      context("instance-index is null", function() {
        it("return true", function() {
          requestObj.query["instance-index"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
        });
      });

      context("instance-index is not an integer", function() {
        it("return false", function() {
          requestObj.query["instance-index"] = "not-integer";
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("instance-index must be an integer");
        });
      });

      context("instance-index is smaller than 0", function() {
        it("return false", function() {
          requestObj.query["instance-index"] = -1;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(false);
          expect(validateResult.message).to.equal("instance-index must be greater than or equal to 0");
        });
      });
    });

  });
});