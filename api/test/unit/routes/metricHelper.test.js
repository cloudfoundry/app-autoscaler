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
    context("validate order-direction", function() {
      context("order-direction is undefined", function() {
        it("return true, order-direction is set to ASC", function() {
          delete requestObj.query["order-direction"];
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters["order"]).to.equal("ASC");
        });

      });
      context("order-direction is null", function() {
        it("return true, order-direction is set to ASC", function() {
          requestObj.query["order-direction"] = null;
          validateResult = helper.parseParameter(requestObj);
          expect(validateResult.valid).to.equal(true);
          expect(validateResult.parameters["order"]).to.equal("ASC");
        });

      });
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
});
